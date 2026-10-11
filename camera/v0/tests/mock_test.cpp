#include "synora_camera/mock.hpp"

#include <cassert>
#include <chrono>
#include <cstdlib>
#include <filesystem>
#include <fstream>
#include <iostream>
#include <stdexcept>
#include <string>
#include <thread>
#include <vector>

#include <arpa/inet.h>
#include <netinet/in.h>
#include <sys/socket.h>
#include <unistd.h>

using namespace synora::camera::mock;

struct TestHTTPServer {
  int socket{-1};
  std::string url;
  std::jthread worker;

  TestHTTPServer(int status, std::string body, std::chrono::milliseconds delay = {}) {
    socket = ::socket(AF_INET, SOCK_STREAM, 0);
    if (socket < 0) throw std::runtime_error("test HTTP socket creation failed");
    int reuse = 1;
    (void)::setsockopt(socket, SOL_SOCKET, SO_REUSEADDR, &reuse, sizeof(reuse));
    sockaddr_in address{};
    address.sin_family = AF_INET;
    address.sin_addr.s_addr = htonl(INADDR_LOOPBACK);
    address.sin_port = 0;
    if (::bind(socket, reinterpret_cast<sockaddr*>(&address), sizeof(address)) != 0 ||
        ::listen(socket, 1) != 0) {
      ::close(socket);
      throw std::runtime_error("test HTTP socket bind/listen failed");
    }
    socklen_t address_size = sizeof(address);
    if (::getsockname(socket, reinterpret_cast<sockaddr*>(&address), &address_size) != 0) {
      ::close(socket);
      throw std::runtime_error("test HTTP socket address failed");
    }
    url = "http://127.0.0.1:" + std::to_string(ntohs(address.sin_port)) + "/vision";
    worker = std::jthread([this, status, body = std::move(body), delay](std::stop_token) {
      const int client = ::accept(socket, nullptr, nullptr);
      if (client < 0) return;
      std::string request;
      std::vector<char> buffer(8192);
      std::size_t expected_size = 0;
      for (;;) {
        const auto count = ::recv(client, buffer.data(), buffer.size(), 0);
        if (count <= 0) break;
        request.append(buffer.data(), static_cast<std::size_t>(count));
        const auto header_end = request.find("\r\n\r\n");
        if (header_end != std::string::npos && expected_size == 0) {
          const auto key = request.find("Content-Length:");
          if (key != std::string::npos) {
            const auto value_start = key + std::string("Content-Length:").size();
            expected_size = static_cast<std::size_t>(std::stoull(request.substr(value_start)));
            expected_size += header_end + 4;
          }
        }
        if (expected_size != 0 && request.size() >= expected_size) break;
      }
      if (delay > std::chrono::milliseconds::zero()) std::this_thread::sleep_for(delay);
      const auto response = "HTTP/1.1 " + std::to_string(status) + " Test\r\nContent-Type: application/json\r\nContent-Length: " +
                            std::to_string(body.size()) + "\r\nConnection: close\r\n\r\n" + body;
      std::size_t sent = 0;
      while (sent < response.size()) {
        const auto count = ::send(client, response.data() + sent, response.size() - sent, MSG_NOSIGNAL);
        if (count <= 0) break;
        sent += static_cast<std::size_t>(count);
      }
      ::close(client);
    });
  }

  ~TestHTTPServer() {
    if (socket >= 0) ::close(socket);
  }
};

std::string unavailable_loopback_url() {
  const int socket = ::socket(AF_INET, SOCK_STREAM, 0);
  if (socket < 0) throw std::runtime_error("unavailable endpoint socket creation failed");
  sockaddr_in address{};
  address.sin_family = AF_INET;
  address.sin_addr.s_addr = htonl(INADDR_LOOPBACK);
  address.sin_port = 0;
  if (::bind(socket, reinterpret_cast<sockaddr*>(&address), sizeof(address)) != 0) {
    ::close(socket);
    throw std::runtime_error("unavailable endpoint bind failed");
  }
  socklen_t address_size = sizeof(address);
  if (::getsockname(socket, reinterpret_cast<sockaddr*>(&address), &address_size) != 0) {
    ::close(socket);
    throw std::runtime_error("unavailable endpoint address failed");
  }
  const auto port = ntohs(address.sin_port);
  ::close(socket); // reserve no listener: curl must observe endpoint-unavailable
  return "http://127.0.0.1:" + std::to_string(port) + "/vision";
}

int main() {
  const auto root = std::filesystem::temp_directory_path() /
                    ("synora-camera-mock-tests-" + std::to_string(::getpid()));
  if (!std::filesystem::create_directory(root))
    throw std::runtime_error("mock test temporary directory already exists");
  const auto artifact = root / "clip.mp4";
  { std::ofstream output(artifact); output << "fixture bytes"; }
  BoundedUploadQueue queue(1);
  assert(queue.enqueue({"clip-1", artifact, 0}));
  assert(!queue.enqueue({"clip-2", artifact, 0})); // saturation rejects without deleting either file
  assert(!queue.enqueue({"clip-1", artifact, 0})); // duplicate identity is idempotently rejected
  auto item = queue.take();
  assert(item && item->clip_id == "clip-1");
  assert(!queue.acknowledge(*item, {true, "clip-other", 202, "queued"}));
  assert(std::filesystem::exists(artifact)); // wrong acknowledgement never erases the artifact
  assert(!queue.acknowledge(*item, {true, "clip-1", 202, "accepted"}));
  assert(std::filesystem::exists(artifact)); // a generic 202 is not the ingress ACK contract
  assert(!queue.acknowledge(*item, {false, {}, 0, "timeout"}));
  assert(std::filesystem::exists(artifact)); // timeout retains the retryable artifact
  assert(!queue.acknowledge(*item, {false, "clip-1", 202, "queued"}));
  assert(std::filesystem::exists(artifact)); // no explicit positive ACK, no deletion
  item->attempts = 1;
  assert(queue.retry(*item, 3));
  item = queue.take();
  assert(item && item->attempts == 1);
  assert(!queue.acknowledge(*item, {true, "clip-1", 503, "queued"}));
  assert(std::filesystem::exists(artifact));
  item->attempts = 2;
  assert(queue.retry(*item, 3));
  item = queue.take();
  assert(item && item->attempts == 2);
  item->attempts = 3;
  assert(!queue.retry(*item, 3)); // retries are bounded and the artifact remains diagnostic evidence
  assert(std::filesystem::exists(artifact));
  item->attempts = 0;
  assert(queue.acknowledge(*item, {true, "clip-1", 202, "queued"}));
  assert(!std::filesystem::exists(artifact));

  const char* old_secret = std::getenv("SYNORA_CAMERA_SECRET_HASH");
  const std::string saved_secret = old_secret == nullptr ? "" : old_secret;
  const bool had_secret = old_secret != nullptr;
  (void)::setenv("SYNORA_CAMERA_SECRET_HASH", "test-verifier-value", 1);
  const auto upload_artifact = root / "upload.mp4";
  { std::ofstream output(upload_artifact, std::ios::binary); output << "fixture-mp4-test-body"; }
  Options options;
  options.camera_id = "camera-mock-test";
  options.data_dir = root;
  options.fixture_clip = upload_artifact;
  options.timeout = std::chrono::milliseconds(100);

  {
    TestHTTPServer server(202, R"({"status":"queued","clip_id":"upload-1"})");
    options.ingest_url = server.url;
    const auto result = post_fixture(options, {"upload-1", upload_artifact, 0});
    assert(result.acknowledged && result.http_status == 202 && result.acknowledged_clip_id == "upload-1");
    server.worker.join();
  }
  {
    options.ingest_url = unavailable_loopback_url();
    const auto result = post_fixture(options, {"upload-1", upload_artifact, 0});
    assert(!result.acknowledged && result.http_status == 0);
    assert(std::filesystem::exists(upload_artifact));
  }
  {
    TestHTTPServer server(503, R"({"status":"unavailable"})");
    options.ingest_url = server.url;
    const auto result = post_fixture(options, {"upload-1", upload_artifact, 0});
    assert(!result.acknowledged && result.http_status == 503);
    server.worker.join();
    assert(std::filesystem::exists(upload_artifact)); // server rejection cannot delete the clip
  }
  {
    TestHTTPServer server(202, R"({"status":"queued","clip_id":"upload-1"})", std::chrono::milliseconds(300));
    options.ingest_url = server.url;
    const auto result = post_fixture(options, {"upload-1", upload_artifact, 0});
    assert(!result.acknowledged);
    server.worker.join();
    assert(std::filesystem::exists(upload_artifact)); // timeout preserves it for retry/restart
  }
  {
    const auto pending_dir = root / "restart-pending";
    std::filesystem::create_directory(pending_dir);
    const auto retryable = pending_dir / "retryable.mp4";
    const auto exhausted = pending_dir / "exhausted.mp4";
    const auto corrupt = pending_dir / "corrupt.mp4";
    { std::ofstream(retryable) << "retryable"; }
    { std::ofstream(retryable.string() + ".attempts") << "2\n"; }
    { std::ofstream(exhausted) << "exhausted"; }
    { std::ofstream(exhausted.string() + ".attempts") << "3\n"; }
    { std::ofstream(corrupt) << "corrupt"; }
    { std::ofstream(corrupt.string() + ".attempts") << "invalid\n"; }
    BoundedUploadQueue restarted_queue(2);
    const auto recovery = recover_pending(restarted_queue, pending_dir, 2);
    assert(recovery.queued == 1 && recovery.exhausted == 1 && recovery.invalid_state == 1);
    const auto recovered_item = restarted_queue.take();
    assert(recovered_item && recovered_item->clip_id == "retryable" && recovered_item->attempts == 2);
    assert(std::filesystem::exists(exhausted) && std::filesystem::exists(corrupt));
  }
  (void)::unsetenv("SYNORA_CAMERA_SECRET_HASH");
  if (had_secret) (void)::setenv("SYNORA_CAMERA_SECRET_HASH", saved_secret.c_str(), 1);
  std::filesystem::remove_all(root);
}
