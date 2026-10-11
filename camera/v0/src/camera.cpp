#include "synora_camera/camera.hpp"

#include <algorithm>
#include <cmath>
#include <fstream>
#include <iomanip>
#include <sstream>
#include <stdexcept>
#include <string_view>
#include <sys/statvfs.h>

namespace synora::camera {
namespace {

float intersection_over_union(const Box& a, const Box& b) {
  const auto left = std::max(a.x, b.x);
  const auto top = std::max(a.y, b.y);
  const auto right = std::min(a.x + a.width, b.x + b.width);
  const auto bottom = std::min(a.y + a.height, b.y + b.height);
  const auto width = std::max(0.0F, right - left);
  const auto height = std::max(0.0F, bottom - top);
  const auto overlap = width * height;
  const auto combined = a.width * a.height + b.width * b.height - overlap;
  return combined > 0.0F ? overlap / combined : 0.0F;
}

float centre_distance(const Box& a, const Box& b) {
  const auto ax = a.x + a.width / 2.0F;
  const auto ay = a.y + a.height / 2.0F;
  const auto bx = b.x + b.width / 2.0F;
  const auto by = b.y + b.height / 2.0F;
  return std::hypot(ax - bx, ay - by);
}

std::string timestamp_ms(TimePoint at) {
  return std::to_string(std::chrono::duration_cast<std::chrono::milliseconds>(
                            at.time_since_epoch())
                            .count());
}

std::string box_json(const Box& box) {
  std::ostringstream output;
  output << std::fixed << std::setprecision(4) << "{\"x\":" << box.x
         << ",\"y\":" << box.y << ",\"width\":" << box.width
         << ",\"height\":" << box.height << '}';
  return output.str();
}

std::uint64_t free_bytes(const std::filesystem::path& path) {
  struct statvfs stats {};
  return statvfs(path.c_str(), &stats) == 0
             ? static_cast<std::uint64_t>(stats.f_bavail) * stats.f_frsize
             : 0U;
}

}  // namespace

struct CameraDaemon::ActiveClip {
  std::string event_id;
  std::uint32_t track_id{};
  TimePoint ends_at;
  std::filesystem::path path;
  std::ofstream file;
};

Tracker::Tracker(std::size_t maximum_tracks) : maximum_tracks_(maximum_tracks) {}

std::vector<Track> Tracker::update(const std::vector<Detection>& detections) {
  std::vector<bool> used(detections.size(), false);
  for (auto& track : tracks_) {
    std::optional<std::size_t> candidate;
    float best_score = -1.0F;
    for (std::size_t index = 0; index < detections.size(); ++index) {
      if (used[index]) continue;
      const float iou = intersection_over_union(track.box, detections[index].box);
      const float distance = centre_distance(track.box, detections[index].box);
      const float score = std::max(iou, 0.20F - distance);
      if ((iou >= 0.15F || distance <= 0.15F) && score > best_score) {
        candidate = index;
        best_score = score;
      }
    }
    if (candidate) {
      const auto& detection = detections[*candidate];
      track.box = detection.box;
      track.probability = detection.human_probability;
      ++track.age_frames;
      track.missing_frames = 0;
      used[*candidate] = true;
    } else {
      ++track.missing_frames;
    }
  }
  tracks_.erase(std::remove_if(tracks_.begin(), tracks_.end(),
                               [](const Track& track) { return track.missing_frames > 3; }),
                tracks_.end());
  for (std::size_t index = 0; index < detections.size() && tracks_.size() < maximum_tracks_; ++index) {
    if (!used[index]) {
      tracks_.push_back(Track{next_id_++, detections[index].box,
                              detections[index].human_probability, 1, 0});
    }
  }
  return tracks_;
}

const std::vector<Track>& Tracker::tracks() const noexcept { return tracks_; }

CameraDaemon::CameraDaemon(Config config)
    : config_(std::move(config)), tracker_(config_.maximum_tracks) {
  if (config_.prebuffer_bytes == 0 || config_.maximum_tracks == 0 ||
      config_.maximum_active_clips == 0) {
    throw std::invalid_argument("camera limits must be non-zero");
  }
  std::filesystem::create_directories(config_.data_directory / "clips");
  std::filesystem::create_directories(config_.data_directory / "outbox");
}

CameraDaemon::~CameraDaemon() = default;

void CameraDaemon::append_prebuffer(const EncodedUnit& unit) {
  prebuffer_size_ += unit.h265.size();
  prebuffer_.push_back(unit);
  const auto oldest_allowed = unit.captured_at - config_.prebuffer_duration;
  while (!prebuffer_.empty() &&
         (prebuffer_size_ > config_.prebuffer_bytes ||
          prebuffer_.front().captured_at < oldest_allowed)) {
    prebuffer_size_ -= prebuffer_.front().h265.size();
    prebuffer_.pop_front();
  }
}

void CameraDaemon::emit(std::string_view json) {
  std::ofstream out(config_.data_directory / "outbox" / "events.jsonl", std::ios::app);
  if (!out) throw std::runtime_error("cannot open durable event outbox");
  out << json << '\n';
  out.flush();
}

void CameraDaemon::start_person_event(const Track& track, TimePoint at) {
  if (clips_.size() >= config_.maximum_active_clips) {
    emit_motion_unknown(at);
    return;
  }
  const auto event_id = config_.camera_id + "-" + std::to_string(++event_sequence_);
  const auto clip_path = config_.data_directory / "clips" / (event_id + ".h265");
  auto clip = std::make_unique<ActiveClip>(ActiveClip{event_id, track.id,
      at + config_.clip_postroll, clip_path, std::ofstream(clip_path, std::ios::binary)});
  if (!clip->file) throw std::runtime_error("cannot open clip spool");
  for (const auto& unit : prebuffer_) {
    clip->file.write(reinterpret_cast<const char*>(unit.h265.data()),
                     static_cast<std::streamsize>(unit.h265.size()));
  }
  emit("{\"schema\":\"synora.camera.v0\",\"event\":\"person_suspect\",\"phase\":\"open\",\"event_id\":\"" +
       event_id + "\",\"camera_id\":\"" + config_.camera_id +
       "\",\"captured_at_ms\":" + timestamp_ms(at) + ",\"track\":{\"id\":" +
       std::to_string(track.id) + ",\"human_probability\":" +
       std::to_string(track.probability) + ",\"box\":" + box_json(track.box) +
       "},\"clip\":{\"codec\":\"h265\",\"prebuffer_seconds\":10,\"postroll_seconds\":20}}" );
  clips_.push_back(std::move(clip));
  person_track_ = track.id;
  pending_motion_.reset();
}

void CameraDaemon::emit_track_update(const Track& track, TimePoint at) {
  if (!person_track_ || *person_track_ != track.id || clips_.empty()) return;
  emit("{\"schema\":\"synora.camera.v0\",\"event\":\"person_suspect\",\"phase\":\"update\",\"event_id\":\"" +
       clips_.front()->event_id + "\",\"camera_id\":\"" + config_.camera_id +
       "\",\"captured_at_ms\":" + timestamp_ms(at) + ",\"track\":{\"id\":" +
       std::to_string(track.id) + ",\"human_probability\":" +
       std::to_string(track.probability) + ",\"box\":" + box_json(track.box) + "}}" );
}

void CameraDaemon::finish_clip(ActiveClip& clip, TimePoint at) {
  clip.file.flush();
  clip.file.close();
  const auto clip_bytes = std::filesystem::file_size(clip.path);
  emit("{\"schema\":\"synora.camera.v0\",\"event\":\"person_suspect\",\"phase\":\"close\",\"event_id\":\"" +
       clip.event_id + "\",\"camera_id\":\"" + config_.camera_id +
       "\",\"captured_at_ms\":" + timestamp_ms(at) + ",\"track_id\":" +
       std::to_string(clip.track_id) + ",\"clip\":{\"path\":\"" + clip.path.string() +
       "\",\"codec\":\"h265\",\"bytes\":" + std::to_string(clip_bytes) + "}}" );
}

void CameraDaemon::emit_motion_unknown(TimePoint at) {
  emit("{\"schema\":\"synora.camera.v0\",\"event\":\"motion_unknown\",\"phase\":\"close\",\"camera_id\":\"" +
       config_.camera_id + "\",\"captured_at_ms\":" + timestamp_ms(at) +
       ",\"context_image\":false,\"clip_requested\":" +
       std::string(config_.motion_clips ? "true" : "false") + "}");
}

void CameraDaemon::process(const FrameInput& input) {
  append_prebuffer(input.encoded);
  for (auto& clip : clips_) {
    clip->file.write(reinterpret_cast<const char*>(input.encoded.h265.data()),
                     static_cast<std::streamsize>(input.encoded.h265.size()));
  }
  if (input.tamper && !tamper_active_) {
    emit("{\"schema\":\"synora.camera.v0\",\"event\":\"tamper\",\"phase\":\"open\",\"camera_id\":\"" +
         config_.camera_id + "\",\"captured_at_ms\":" + timestamp_ms(input.encoded.captured_at) + "}");
    tamper_active_ = true;
  } else if (!input.tamper && tamper_active_) {
    emit("{\"schema\":\"synora.camera.v0\",\"event\":\"tamper\",\"phase\":\"close\",\"camera_id\":\"" +
         config_.camera_id + "\",\"captured_at_ms\":" + timestamp_ms(input.encoded.captured_at) + "}");
    tamper_active_ = false;
  }
  if ((input.motion || input.pir) && !pending_motion_ && !person_track_) {
    pending_motion_ = input.encoded.captured_at;
  }
  const auto tracks = tracker_.update(input.detections);
  const auto person = std::find_if(tracks.begin(), tracks.end(), [](const Track& track) {
    return track.missing_frames == 0 && track.probability >= 0.65F;
  });
  if (person != tracks.end() && !person_track_) start_person_event(*person, input.encoded.captured_at);
  if (person != tracks.end()) emit_track_update(*person, input.encoded.captured_at);
  if (pending_motion_ && input.encoded.captured_at - *pending_motion_ >= config_.motion_grace) {
    emit_motion_unknown(input.encoded.captured_at);
    pending_motion_.reset();
  }
  for (auto iterator = clips_.begin(); iterator != clips_.end();) {
    if (input.encoded.captured_at >= (*iterator)->ends_at) {
      finish_clip(**iterator, input.encoded.captured_at);
      iterator = clips_.erase(iterator);
      person_track_.reset();
    } else {
      ++iterator;
    }
  }
}

void CameraDaemon::publish_health(std::string_view reason) {
  const auto bytes = free_bytes(config_.data_directory);
  emit("{\"schema\":\"synora.camera.v0\",\"event\":\"health\",\"phase\":\"update\",\"camera_id\":\"" +
       config_.camera_id + "\",\"reason\":\"" + std::string(reason) +
       "\",\"prebuffer_bytes\":" + std::to_string(prebuffer_size_) +
       ",\"active_clips\":" + std::to_string(clips_.size()) +
       ",\"storage_free_bytes\":" + std::to_string(bytes) +
       ",\"sensor\":\"ok\",\"encoder\":\"ok\",\"network\":\"unknown\",\"update_policy\":\"signed-only\"}");
}

std::size_t CameraDaemon::prebuffer_size() const noexcept { return prebuffer_size_; }
std::size_t CameraDaemon::active_clip_count() const noexcept { return clips_.size(); }

SimulatedPipeline::SimulatedPipeline(std::chrono::milliseconds period, int person_start_second,
                                     int person_end_second, int tamper_second)
    : period_(period), person_start_second_(person_start_second),
      person_end_second_(person_end_second), tamper_second_(tamper_second) {}

std::optional<FrameInput> SimulatedPipeline::next() {
  const auto captured_at = started_at_ + period_ * frame_;
  const auto seconds = static_cast<int>((period_ * frame_).count() / 1000);
  const bool person = seconds >= person_start_second_ && seconds < person_end_second_;
  FrameInput result{{captured_at, frame_ % 25 == 0,
                     std::vector<std::uint8_t>(80U * 1024U, static_cast<std::uint8_t>(frame_ & 0xffU))},
                    person || seconds == 2, false, seconds == tamper_second_, {}};
  if (person) {
    const float x = 0.10F + static_cast<float>(frame_ % 60U) / 500.0F;
    result.detections.push_back(Detection{{x, 0.22F, 0.22F, 0.50F}, 0.86F});
  }
  ++frame_;
  return result;
}

}  // namespace synora::camera
