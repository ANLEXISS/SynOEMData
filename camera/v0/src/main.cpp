#include "synora_camera/camera.hpp"

#include <chrono>
#include <cstdlib>
#include <iostream>
#include <stdexcept>
#include <string>
#include <thread>

int main(int argc, char** argv) {
  try {
    int duration_seconds = 35;
    synora::camera::Config config;
    for (int index = 1; index < argc; ++index) {
      const std::string option = argv[index];
      if (option == "--duration" && index + 1 < argc) duration_seconds = std::stoi(argv[++index]);
      else if (option == "--data-dir" && index + 1 < argc) config.data_directory = argv[++index];
      else if (option == "--camera-id" && index + 1 < argc) config.camera_id = argv[++index];
      else if (option == "--motion-clips") config.motion_clips = true;
      else if (option == "--help") {
        std::cout << "synora-camera --duration SECONDS --data-dir PATH --camera-id ID [--motion-clips]\n";
        return EXIT_SUCCESS;
      } else throw std::invalid_argument("unknown or incomplete option: " + option);
    }
    if (duration_seconds < 1) throw std::invalid_argument("duration must be positive");
    constexpr auto frame_period = std::chrono::milliseconds(200);
    synora::camera::CameraDaemon daemon(config);
    synora::camera::SimulatedPipeline pipeline(frame_period, 4, 10, -1);
    daemon.publish_health("startup");
    for (int frame = 0; frame < duration_seconds * 5; ++frame) {
      daemon.process(*pipeline.next());
      std::this_thread::sleep_for(frame_period);
    }
    daemon.publish_health("shutdown");
  } catch (const std::exception& error) {
    std::cerr << "synora-camera: " << error.what() << '\n';
    return EXIT_FAILURE;
  }
  return EXIT_SUCCESS;
}
