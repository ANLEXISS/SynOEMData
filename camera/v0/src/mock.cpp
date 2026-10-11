#include "synora_camera/mock.hpp"
#include "synora_camera/camera.hpp"

#include <array>
#include <atomic>
#include <cctype>
#include <cerrno>
#include <chrono>
#include <cstdlib>
#include <fcntl.h>
#include <fstream>
#include <iostream>
#include <regex>
#include <spawn.h>
#include <sstream>
#include <stdexcept>
#include <thread>

#include <sys/wait.h>
#include <unistd.h>

extern char** environ;

namespace synora::camera::mock {
namespace {

std::string read_all(const std::filesystem::path& path) {
  std::ifstream input(path);
  if (!input) return {};
  return {std::istreambuf_iterator<char>(input), std::istreambuf_iterator<char>()};
}

std::optional<unsigned> load_attempts(const std::filesystem::path& artifact) {
  const auto sidecar = std::filesystem::path(artifact.string() + ".attempts");
  std::ifstream input(sidecar);
  if (!input && !std::filesystem::exists(sidecar)) return 0;
  if (!input) return std::nullopt;
  unsigned value = 0;
  if (!(input >> value)) return std::nullopt;
  input >> std::ws;
  if (!input.eof()) return std::nullopt;
  return value;
}

void persist_attempts(const std::filesystem::path& artifact, unsigned attempts) {
  const auto sidecar = std::filesystem::path(artifact.string() + ".attempts");
  const auto temporary = std::filesystem::path(sidecar.string() + ".tmp");
  {
    std::ofstream output(temporary, std::ios::trunc);
    output << attempts << '\n';
    if (!output) throw std::runtime_error("cannot persist upload retry count");
  }
  std::filesystem::rename(temporary, sidecar);
}

std::string shell_free_capture(const std::vector<std::string>& arguments,
                               const std::filesystem::path& stdin_path = {}) {
  int output_pipe[2];
  if (::pipe(output_pipe) != 0) throw std::runtime_error("cannot create helper pipe");
  posix_spawn_file_actions_t actions;
  if (posix_spawn_file_actions_init(&actions) != 0) {
    ::close(output_pipe[0]);
    ::close(output_pipe[1]);
    throw std::runtime_error("cannot prepare HTTP helper");
  }
  int action_error = posix_spawn_file_actions_addclose(&actions, output_pipe[0]);
  if (action_error == 0)
    action_error = posix_spawn_file_actions_adddup2(&actions, output_pipe[1], STDOUT_FILENO);
  if (action_error == 0)
    action_error = posix_spawn_file_actions_addclose(&actions, output_pipe[1]);
  if (action_error == 0 && !stdin_path.empty())
    action_error = posix_spawn_file_actions_addopen(&actions, STDIN_FILENO,
                                                    stdin_path.c_str(), O_RDONLY, 0);
  if (action_error != 0) {
    posix_spawn_file_actions_destroy(&actions);
    ::close(output_pipe[0]);
    ::close(output_pipe[1]);
    throw std::runtime_error("cannot prepare HTTP helper actions");
  }
  std::vector<char*> argv;
  argv.reserve(arguments.size() + 1);
  for (const auto& argument : arguments) argv.push_back(const_cast<char*>(argument.c_str()));
  argv.push_back(nullptr);
  pid_t child{};
  const int spawn_error = posix_spawnp(&child, argv.front(), &actions, nullptr, argv.data(), ::environ);
  posix_spawn_file_actions_destroy(&actions);
  ::close(output_pipe[1]);
  if (spawn_error != 0) {
    ::close(output_pipe[0]);
    throw std::runtime_error("cannot start HTTP helper");
  }
  std::string output;
  std::array<char, 512> buffer{};
  for (;;) {
    const auto count = ::read(output_pipe[0], buffer.data(), buffer.size());
    if (count <= 0) break;
    output.append(buffer.data(), static_cast<std::size_t>(count));
  }
  ::close(output_pipe[0]);
  int status = 0;
  while (::waitpid(child, &status, 0) < 0 && errno == EINTR) {}
  if (!WIFEXITED(status) || WEXITSTATUS(status) != 0)
    throw std::runtime_error("HTTP helper failed");
  return output;
}

std::string sha256_file(const std::filesystem::path& path) {
  const auto output = shell_free_capture({"openssl", "dgst", "-sha256", "-r", path.string()});
  static const std::regex digest(R"(^([0-9a-fA-F]{64})\s)");
  std::smatch match;
  if (!std::regex_search(output, match, digest)) throw std::runtime_error("cannot hash fixture clip");
  return match[1].str();
}

std::string hmac_sha256(std::string_view payload, std::string_view secret_hash,
                        const std::filesystem::path& scratch) {
  const auto payload_file = scratch / "signature-input";
  {
    std::ofstream output(payload_file, std::ios::binary | std::ios::trunc);
    output.write(payload.data(), static_cast<std::streamsize>(payload.size()));
    if (!output) throw std::runtime_error("cannot prepare signature input");
  }
  const auto output = shell_free_capture(
      {"openssl", "dgst", "-sha256", "-mac", "HMAC", "-macopt",
       "key:" + std::string(secret_hash)}, payload_file);
  std::filesystem::remove(payload_file);
  const auto separator = output.find('=');
  if (separator == std::string::npos) throw std::runtime_error("cannot sign ingress request");
  auto signature = output.substr(separator + 1);
  while (!signature.empty() && std::isspace(static_cast<unsigned char>(signature.back()))) signature.pop_back();
  return signature;
}

std::string event_value(std::string_view line, std::string_view key) {
  const std::regex pattern("\\\"" + std::string(key) + "\\\":\\\"([^\\\"]*)\\\"");
  std::smatch match;
  const std::string text(line);
  return std::regex_search(text, match, pattern) ? match[1].str() : std::string{};
}

bool safe_id(std::string_view value) {
  return !value.empty() && value.find_first_not_of("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_.") == std::string_view::npos &&
         value.find("..") == std::string_view::npos;
}

}  // namespace

BoundedUploadQueue::BoundedUploadQueue(std::size_t capacity) : capacity_(capacity) {
  if (capacity == 0) throw std::invalid_argument("upload queue capacity must be positive");
}

bool BoundedUploadQueue::enqueue(UploadItem item) {
  std::lock_guard lock(mutex_);
  if (!safe_id(item.clip_id) || !std::filesystem::is_regular_file(item.artifact) ||
      items_.size() >= capacity_ || known_ids_.contains(item.clip_id)) return false;
  known_ids_.insert(item.clip_id);
  items_.push_back(std::move(item));
  return true;
}

std::optional<UploadItem> BoundedUploadQueue::take() {
  std::lock_guard lock(mutex_);
  if (items_.empty()) return std::nullopt;
  auto item = std::move(items_.front());
  items_.erase(items_.begin());
  return item;
}

bool BoundedUploadQueue::retry(UploadItem item, unsigned maximum_attempts) {
  std::lock_guard lock(mutex_);
  if (item.attempts >= maximum_attempts || items_.size() >= capacity_) return false;
  items_.push_back(std::move(item));
  return true;
}

bool BoundedUploadQueue::acknowledge(const UploadItem& item, const UploadResult& result) {
  std::lock_guard lock(mutex_);
  if (!result.acknowledged || result.http_status != 202 || result.reason != "queued" ||
      result.acknowledged_clip_id != item.clip_id)
    return false;
  std::error_code error;
  const bool removed = std::filesystem::remove(item.artifact, error);
  if (error || !removed) return false;
  std::filesystem::remove(item.artifact.string() + ".attempts", error);
  known_ids_.erase(item.clip_id);
  return true;
}

std::size_t BoundedUploadQueue::size() const {
  std::lock_guard lock(mutex_);
  return items_.size();
}

bool BoundedUploadQueue::retained(const std::filesystem::path& artifact) const {
  return std::filesystem::exists(artifact);
}

RecoveryReport recover_pending(BoundedUploadQueue& queue,
                               const std::filesystem::path& pending_dir,
                               unsigned maximum_retries) {
  RecoveryReport report;
  for (const auto& entry : std::filesystem::directory_iterator(pending_dir)) {
    if (!entry.is_regular_file() || entry.path().extension() != ".mp4") continue;
    const auto id = entry.path().stem().string();
    const auto attempts = load_attempts(entry.path());
    if (!attempts) {
      ++report.invalid_state;
    } else if (*attempts > maximum_retries) {
      ++report.exhausted;
    } else if (queue.enqueue({id, entry.path(), *attempts})) {
      ++report.queued;
    } else {
      ++report.rejected;
    }
  }
  return report;
}

UploadResult post_fixture(const Options& options, const UploadItem& item) {
  const char* secret = std::getenv("SYNORA_CAMERA_SECRET_HASH");
  if (secret == nullptr || *secret == '\0') return {false, {}, 0, "missing_SYNORA_CAMERA_SECRET_HASH"};
  const auto scratch = options.data_dir / "http";
  std::filesystem::create_directories(scratch);
  const auto checksum = sha256_file(item.artifact);
  const auto timestamp = std::to_string(std::chrono::duration_cast<std::chrono::seconds>(
      std::chrono::system_clock::now().time_since_epoch()).count());
  const auto signature = hmac_sha256(options.camera_id + timestamp + checksum, secret, scratch);
  const auto response_path = scratch / (item.clip_id + ".response.json");
  std::vector<std::string> arguments{
      "curl", "--silent", "--show-error", "--max-time",
      std::to_string(std::max(0.01, std::chrono::duration<double>(options.timeout).count())),
      "--output", response_path.string(), "--write-out", "%{http_code}", "--request", "POST",
      "--header", "X-Synora-Device: " + options.camera_id,
      "--header", "X-Synora-Timestamp: " + timestamp,
      "--header", "X-Synora-Signature: " + signature,
      "--header", "X-Synora-Clip-ID: " + item.clip_id,
      "--header", "X-Synora-Clip-Container: mp4",
      "--header", "X-Synora-Clip-Checksum: " + checksum,
      "--header", "X-Synora-Simulated-Camera: true",
      "--header", "Expect:",
      "--form", "clip=@" + item.artifact.string() + ";type=video/mp4;filename=" + item.clip_id + ".mp4",
      "--form-string", "simulated_camera=true",
      "--form-string", "trigger_reason=person_suspect.close",
      options.ingest_url};
  if (options.scenario == "timeout") {
    arguments[4] = "0.001";
  }
  try {
    const auto code = shell_free_capture(arguments);
    const auto body = read_all(response_path);
    std::filesystem::remove(response_path);
    const auto ack_id = event_value(body, "clip_id");
    const auto ack_status = event_value(body, "status");
    const int http_status = code.size() == 3 ? std::stoi(code) : 0;
    const bool acknowledged = http_status == 202 && ack_status == "queued" && ack_id == item.clip_id;
    return {acknowledged, ack_id, http_status,
            acknowledged ? "queued" : (http_status == 202 ? "ack_mismatch" : "http_rejected")};
  } catch (const std::exception& error) {
    return {false, {}, 0, error.what()};
  }
}

int run(const Options& options) {
  if (!safe_id(options.camera_id)) throw std::invalid_argument("--camera-id is missing or unsafe");
  if (options.data_dir.empty() || options.fixture_clip.empty() || options.ingest_url.empty())
    throw std::invalid_argument("--data-dir, --fixture-clip and --ingest-url are required");
  if (options.duration < std::chrono::seconds(23))
    throw std::invalid_argument("duration must be at least 23 seconds to finalize the V0 postroll clip");
  if (!std::filesystem::is_regular_file(options.fixture_clip)) throw std::invalid_argument("fixture clip not found");
  if (options.scenario != "success" && options.scenario != "timeout" && options.scenario != "offline" &&
      options.scenario != "duplicate" && options.scenario != "queue-full") throw std::invalid_argument("invalid --scenario");
  const auto probe = shell_free_capture({"ffprobe", "-v", "error", "-select_streams", "v:0",
                                         "-show_entries", "stream=codec_name", "-of", "csv=p=0",
                                         options.fixture_clip.string()});
  if (probe.empty()) throw std::invalid_argument("fixture has no decodable video stream");
  std::filesystem::create_directories(options.data_dir / "pending");
  std::filesystem::create_directories(options.data_dir / "outbox");
  {
    std::ofstream marker(options.data_dir / "mock-run.json", std::ios::trunc);
    marker << "{\"simulated_camera\":true,\"camera_id\":\"" << options.camera_id
           << "\",\"scenario\":\"" << options.scenario << "\",\"fixture_sha256\":\""
           << sha256_file(options.fixture_clip) << "\"}\n";
    if (!marker) throw std::runtime_error("cannot write mock identity marker");
  }
  std::cout << "{\"simulated_camera\":true,\"camera_id\":\"" << options.camera_id
            << "\",\"scenario\":\"" << options.scenario << "\"}" << std::endl;

  const auto capacity = options.scenario == "queue-full" ? 1U : options.queue_capacity;
  BoundedUploadQueue queue(capacity);
  const auto recovered = recover_pending(queue, options.data_dir / "pending", options.retries);
  if (recovered.exhausted > 0 || recovered.invalid_state > 0 || recovered.rejected > 0)
    std::cerr << "pending_recovery queued=" << recovered.queued << " exhausted=" << recovered.exhausted
              << " invalid_state=" << recovered.invalid_state << " rejected=" << recovered.rejected
              << " retained=true\n";
  Config camera_config;
  camera_config.camera_id = options.camera_id;
  camera_config.data_directory = options.data_dir;
  CameraDaemon daemon(camera_config);
  SimulatedPipeline pipeline(std::chrono::milliseconds(200), 1, 2, -1);
  std::atomic_bool producer_done{false};
  const auto started = std::chrono::steady_clock::now();
  std::jthread producer([&](std::stop_token stop) {
    daemon.publish_health("startup");
    const auto deadline = std::chrono::steady_clock::now() + options.duration;
    auto next_health = std::chrono::steady_clock::now() + std::chrono::seconds(5);
    while (!stop.stop_requested() && std::chrono::steady_clock::now() < deadline) {
      if (auto frame = pipeline.next()) daemon.process(*frame);
      if (std::chrono::steady_clock::now() >= next_health) {
        daemon.publish_health("periodic");
        next_health = std::chrono::steady_clock::now() + std::chrono::seconds(5);
      }
      std::this_thread::sleep_for(std::chrono::milliseconds(200));
    }
    daemon.publish_health("shutdown");
    producer_done.store(true);
  });

  const auto events_path = options.data_dir / "outbox" / "events.jsonl";
  std::error_code offset_error;
  std::uintmax_t offset = std::filesystem::exists(events_path, offset_error)
                              ? std::filesystem::file_size(events_path, offset_error)
                              : 0;
  std::unordered_set<std::string> dispatched;
  const auto monitor_grace = (options.scenario == "offline" || options.scenario == "queue-full")
                                 ? std::chrono::seconds(2)
                                 : std::chrono::seconds(10);
  const auto monitor_deadline = started + options.duration + monitor_grace;
  unsigned acknowledgements = 0;
  bool offline_reported = false;
  while (!producer_done.load() || queue.size() > 0) {
    std::this_thread::sleep_for(std::chrono::milliseconds(100));
    std::error_code error;
    if (std::filesystem::exists(events_path, error)) {
      const auto size = std::filesystem::file_size(events_path, error);
      if (!error && size >= offset) {
        std::ifstream events(events_path, std::ios::binary);
        events.seekg(static_cast<std::streamoff>(offset));
        std::string line;
        while (std::getline(events, line)) {
          offset = static_cast<std::uintmax_t>(events.tellg());
          if (event_value(line, "event") != "person_suspect" || event_value(line, "phase") != "close") continue;
          const auto event_id = event_value(line, "event_id");
          if (event_id.empty()) continue;
          if (dispatched.contains(event_id)) continue;
          dispatched.insert(event_id);
          const auto artifact = options.data_dir / "pending" / (event_id + ".mp4");
          if (!std::filesystem::exists(artifact))
            std::filesystem::copy_file(options.fixture_clip, artifact);
          if (!queue.enqueue({event_id, artifact, 0})) {
            std::cerr << "upload_rejected queue_full clip_id=" << event_id << " retained=" << artifact << '\n';
            continue;
          }
          if (options.scenario == "duplicate" && !queue.enqueue({event_id, artifact, 0}))
            std::cerr << "duplicate_suppressed clip_id=" << event_id << '\n';
          if (options.scenario == "queue-full") {
            const auto second = options.data_dir / "pending" / (event_id + "-overflow.mp4");
            std::filesystem::copy_file(options.fixture_clip, second, std::filesystem::copy_options::overwrite_existing);
            if (!queue.enqueue({event_id + "-overflow", second, 0}))
              std::cerr << "upload_rejected queue_full clip_id=" << event_id << "-overflow retained=" << second << '\n';
          }
        }
        const auto end = events.tellg();
        offset = end >= 0 ? static_cast<std::uintmax_t>(end) : size;
      }
    }
    if (queue.size() > 0 && options.scenario != "offline" && options.scenario != "queue-full") {
      auto item = queue.take();
      if (item) {
        bool done = false;
        do {
          ++item->attempts;
          persist_attempts(item->artifact, item->attempts);
          const auto result = post_fixture(options, *item);
          if (queue.acknowledge(*item, result)) {
            std::cout << "upload_ack clip_id=" << item->clip_id << " http=202 status=queued simulated_camera=true\n";
            ++acknowledgements;
            done = true;
          } else {
            std::cerr << "upload_failed clip_id=" << item->clip_id << " http=" << result.http_status
                      << " reason=" << result.reason << " retained=true\n";
            if (!queue.retry(*item, options.retries + 1)) break;
            item = queue.take();
            if (!item) break;
          }
        } while (!done);
      }
    } else if (queue.size() > 0 && options.scenario == "offline" && !offline_reported) {
      std::cerr << "network_offline simulated=true pending=" << queue.size() << " retained=true\n";
      offline_reported = true;
    }
    if (std::chrono::steady_clock::now() >= monitor_deadline) {
      producer.request_stop();
      producer.join();
      break;
    }
  }
  if (producer.joinable()) { producer.request_stop(); producer.join(); }
  if (options.scenario == "success" && acknowledgements == 0) return 1;
  return 0;
}

}  // namespace synora::camera::mock
