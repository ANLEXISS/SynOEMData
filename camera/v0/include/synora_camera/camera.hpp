#pragma once

#include <chrono>
#include <cstdint>
#include <deque>
#include <filesystem>
#include <memory>
#include <optional>
#include <string>
#include <string_view>
#include <vector>

namespace synora::camera {

// Camera events cross the network, so the event timestamp is always UTC Unix
// time. The Rockchip adapter will map its monotonic PTS onto this clock.
using Clock = std::chrono::system_clock;
using TimePoint = Clock::time_point;

struct Box {
  float x{};
  float y{};
  float width{};
  float height{};
};

struct Detection {
  Box box;
  float human_probability{};
};

struct EncodedUnit {
  TimePoint captured_at;
  bool keyframe{};
  std::vector<std::uint8_t> h265;
};

struct FrameInput {
  EncodedUnit encoded;
  bool motion{};
  bool pir{};
  bool tamper{};
  std::vector<Detection> detections;
};

struct Track {
  std::uint32_t id{};
  Box box;
  float probability{};
  std::uint32_t age_frames{};
  std::uint32_t missing_frames{};
};

struct Config {
  std::string camera_id{"camera-proto-01"};
  std::filesystem::path data_directory{"./runtime"};
  std::size_t prebuffer_bytes{4U * 1024U * 1024U};
  std::chrono::seconds prebuffer_duration{10};
  std::chrono::seconds clip_postroll{20};
  std::chrono::milliseconds inference_interval{200};
  std::chrono::seconds motion_grace{1};
  std::size_t maximum_tracks{4};
  std::size_t maximum_active_clips{2};
  bool motion_clips{false};
};

// Fixed interfaces for the EM1103B integration. The production implementation
// owns the Rockchip ISP/VENC/RKNN objects; it must not change event semantics.
class HardwarePipeline {
 public:
  virtual ~HardwarePipeline() = default;
  virtual std::optional<FrameInput> next() = 0;
};

class Tracker {
 public:
  explicit Tracker(std::size_t maximum_tracks);
  std::vector<Track> update(const std::vector<Detection>& detections);
  const std::vector<Track>& tracks() const noexcept;

 private:
  std::size_t maximum_tracks_;
  std::uint32_t next_id_{1};
  std::vector<Track> tracks_;
};

class CameraDaemon {
 public:
  explicit CameraDaemon(Config config);
  ~CameraDaemon();

  void process(const FrameInput& input);
  void publish_health(std::string_view reason = "periodic");
  std::size_t prebuffer_size() const noexcept;
  std::size_t active_clip_count() const noexcept;

 private:
  struct ActiveClip;
  void append_prebuffer(const EncodedUnit& unit);
  void start_person_event(const Track& track, TimePoint at);
  void finish_clip(ActiveClip& clip, TimePoint at);
  void emit(std::string_view json);
  void emit_track_update(const Track& track, TimePoint at);
  void emit_motion_unknown(TimePoint at);
  Config config_;
  Tracker tracker_;
  std::deque<EncodedUnit> prebuffer_;
  std::size_t prebuffer_size_{};
  std::vector<std::unique_ptr<ActiveClip>> clips_;
  std::optional<TimePoint> pending_motion_;
  std::optional<std::uint32_t> person_track_;
  bool tamper_active_{};
  std::uint64_t event_sequence_{};
  std::chrono::seconds last_health_second_{-1};
};

// Development-only producer. It creates H.265-shaped access units and human
// probability results, but never creates decoded frames or recognises anyone.
class SimulatedPipeline final : public HardwarePipeline {
 public:
  SimulatedPipeline(std::chrono::milliseconds period, int person_start_second,
                    int person_end_second, int tamper_second);
  std::optional<FrameInput> next() override;

 private:
  std::chrono::milliseconds period_;
  TimePoint started_at_{Clock::now()};
  std::uint64_t frame_{};
  int person_start_second_;
  int person_end_second_;
  int tamper_second_;
};

}  // namespace synora::camera
