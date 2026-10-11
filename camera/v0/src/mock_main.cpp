#include "synora_camera/mock.hpp"

#include <cstdlib>
#include <iostream>
#include <stdexcept>
#include <string>

namespace {
void usage(std::ostream& output) {
  output << "synora-camera-mock --camera-id ID --data-dir PATH --fixture-clip FILE "
            "--ingest-url URL [--duration SECONDS] "
            "[--scenario success|timeout|offline|duplicate|queue-full]\n"
            "Authentication: SYNORA_CAMERA_SECRET_HASH must contain the central device "
            "transport verifier.\n"
            "Every run is simulated; fixture MP4 is sent instead of the synthetic H.265 spool.\n";
}
}

int main(int argc, char** argv) {
  synora::camera::mock::Options options;
  try {
    for (int index = 1; index < argc; ++index) {
      const std::string name = argv[index];
      if (name == "--help") {
        usage(std::cout);
        return EXIT_SUCCESS;
      }
      if (index + 1 >= argc) throw std::invalid_argument("missing value for " + name);
      const std::string value = argv[++index];
      if (name == "--camera-id") options.camera_id = value;
      else if (name == "--data-dir") options.data_dir = value;
      else if (name == "--fixture-clip") options.fixture_clip = value;
      else if (name == "--ingest-url") options.ingest_url = value;
      else if (name == "--duration") options.duration = std::chrono::seconds(std::stoll(value));
      else if (name == "--scenario") options.scenario = value;
      else throw std::invalid_argument("unknown option: " + name);
    }
    if (options.duration <= std::chrono::seconds::zero()) throw std::invalid_argument("duration must be positive");
    return synora::camera::mock::run(options);
  } catch (const std::exception& error) {
    std::cerr << "synora-camera-mock: " << error.what() << '\n';
    usage(std::cerr);
    return EXIT_FAILURE;
  }
}
