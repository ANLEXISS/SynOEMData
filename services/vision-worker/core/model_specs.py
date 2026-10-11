"""Explicit metadata for model inputs and outputs already used by Vision."""

from .inference_backend import ModelMetadata


PERSON_YOLO = ModelMetadata(
    model_id="yolov8-person",
    task="person-detection",
    input_shape=(1, 640, 640, 3),
    input_dtype="float32",
    input_layout="NHWC",
    output_representation="YOLOv8 detection rows: [1,84,8400] or row-major",
    runtime_data_format="nhwc",
)

SCRFD_FACE = ModelMetadata(
    model_id="scrfd-face-detection",
    task="face-detection-and-landmarks",
    input_shape=(1, 3, 640, 640),
    input_dtype="float32",
    input_layout="NCHW",
    output_representation="SCRFD stride 8/16/32 scores, boxes, optional landmarks",
)

ARCFACE_EMBEDDING = ModelMetadata(
    model_id="arcface-w600k-r50",
    task="face-embedding",
    input_shape=(1, 3, 112, 112),
    input_dtype="float32",
    input_layout="NCHW",
    output_representation="normalized 512-float embedding",
)

RTMPOSE_POSE = ModelMetadata(
    model_id="rtmpose-s",
    task="pose-enrichment",
    input_shape=(1, 3, 256, 192),
    input_dtype="float32",
    input_layout="NCHW",
    output_representation="SimCC x/y tensors for 17 keypoints",
)

YOLOV8_POSE = ModelMetadata(
    model_id="yolov8n-pose",
    task="pose-qualification",
    input_shape=(1, 640, 640, 3),
    input_dtype="uint8",
    input_layout="NHWC",
    output_representation="three DFL detection heads and 17x3 keypoint head",
    runtime_data_format="nhwc",
)
