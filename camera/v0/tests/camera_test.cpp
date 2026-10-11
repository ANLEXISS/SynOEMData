#include "synora_camera/camera.hpp"

#include <cassert>
#include <chrono>
#include <filesystem>
#include <fstream>
#include <stdexcept>
#include <string>
#include <unistd.h>

namespace {
synora::camera::FrameInput frame(synora::camera::TimePoint at, bool motion,
                                 std::vector<synora::camera::Detection> detections = {}) {
  return {{at, true, std::vector<std::uint8_t>(1024U, 0x2a)}, motion, false, false,
          std::move(detections)};
}
}  // namespace

int main() {
  const auto temporary = std::filesystem::temp_directory_path() /
                         ("synora-camera-test-" + std::to_string(::getpid()));
  if (!std::filesystem::create_directory(temporary))
    throw std::runtime_error("camera test temporary directory already exists");
  synora::camera::Config config;
  config.data_directory = temporary;
  config.prebuffer_bytes = 3U * 1024U;
  config.prebuffer_duration = std::chrono::seconds(10);
  config.clip_postroll = std::chrono::seconds(1);
  synora::camera::CameraDaemon daemon(config);
  const auto start = synora::camera::TimePoint{};
  for (int index = 0; index < 5; ++index) daemon.process(frame(start + std::chrono::milliseconds(index * 200), false));
  assert(daemon.prebuffer_size() <= config.prebuffer_bytes);
  daemon.process(frame(start + std::chrono::seconds(1), true,
                       {{{0.10F, 0.20F, 0.20F, 0.40F}, 0.90F}}));
  assert(daemon.active_clip_count() == 1);
  daemon.process(frame(start + std::chrono::seconds(3), false));
  assert(daemon.active_clip_count() == 0);
  std::ifstream events(temporary / "outbox" / "events.jsonl");
  const std::string json((std::istreambuf_iterator<char>(events)), std::istreambuf_iterator<char>());
  assert(json.find("\"event\":\"person_suspect\"") != std::string::npos);
  assert(json.find("\"phase\":\"open\"") != std::string::npos);
  assert(json.find("\"phase\":\"close\"") != std::string::npos);
  synora::camera::CameraDaemon motion_daemon(config);
  motion_daemon.process(frame(start, true));
  motion_daemon.process(frame(start + std::chrono::seconds(2), false));
  std::ifstream motion_events(temporary / "outbox" / "events.jsonl");
  const std::string motion_json((std::istreambuf_iterator<char>(motion_events)), std::istreambuf_iterator<char>());
  assert(motion_json.find("\"event\":\"motion_unknown\"") != std::string::npos);
  std::filesystem::remove_all(temporary);
}
