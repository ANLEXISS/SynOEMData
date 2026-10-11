#pragma once

#include <chrono>
#include <condition_variable>
#include <filesystem>
#include <functional>
#include <mutex>
#include <optional>
#include <string>
#include <unordered_set>
#include <vector>

namespace synora::camera::mock {

struct UploadItem {
  std::string clip_id;
  std::filesystem::path artifact;
  unsigned attempts{};
};

struct UploadResult {
  bool acknowledged{};
  std::string acknowledged_clip_id;
  int http_status{};
  std::string reason;
};

class BoundedUploadQueue {
 public:
  explicit BoundedUploadQueue(std::size_t capacity);
  bool enqueue(UploadItem item);
  std::optional<UploadItem> take();
  bool retry(UploadItem item, unsigned maximum_attempts);
  bool acknowledge(const UploadItem& item, const UploadResult& result);
  std::size_t size() const;
  bool retained(const std::filesystem::path& artifact) const;

 private:
  const std::size_t capacity_;
  mutable std::mutex mutex_;
  std::vector<UploadItem> items_;
  std::unordered_set<std::string> known_ids_;
};

struct RecoveryReport {
  std::size_t queued{};
  std::size_t exhausted{};
  std::size_t invalid_state{};
  std::size_t rejected{};
};

RecoveryReport recover_pending(BoundedUploadQueue& queue,
                               const std::filesystem::path& pending_dir,
                               unsigned maximum_retries);

struct Options {
  std::string camera_id;
  std::filesystem::path data_dir;
  std::filesystem::path fixture_clip;
  std::string ingest_url;
  std::chrono::seconds duration{35};
  std::string scenario{"success"};
  std::chrono::milliseconds timeout{5000};
  unsigned retries{2};
  std::size_t queue_capacity{4};
};

UploadResult post_fixture(const Options& options, const UploadItem& item);
int run(const Options& options);

}  // namespace synora::camera::mock
