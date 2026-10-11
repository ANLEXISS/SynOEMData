#!/usr/bin/env python3
"""Isolated standard-library I1 next-state learning experiment.

This is a research proof of concept, not a production encoder or runtime.
It generates synthetic occupancy transitions, splits complete homes, fits a
small one-hidden-layer MLP, and compares it with majority and last-state
predictors. No external packages or project dependencies are required.
"""

from __future__ import annotations

import hashlib
import json
import math
import random


DATA_SEED = 104729
MODEL_SEED = 130363
HOME_COUNT = 60
STEPS_PER_HOME = 128
TRAIN_HOMES = 36
VALIDATION_HOMES = 12
HIDDEN_SIZE = 6
MAX_EPOCHS = 180
BATCH_SIZE = 32
LEARNING_RATE = 0.08
L2 = 0.0001
PATIENCE = 18


def generate_data():
    rng = random.Random(DATA_SEED)
    homes = []
    records = []
    for home_index in range(HOME_COUNT):
        home_id = f"synthetic-home-{home_index:03d}"
        homes.append(home_id)
        door_rate = rng.uniform(0.18, 0.62)
        motion_rate = rng.uniform(0.28, 0.72)
        state = rng.randrange(2)
        for step in range(STEPS_PER_HOME):
            door_open = int(rng.random() < door_rate)
            motion = int(rng.random() < motion_rate)
            next_state = state ^ int(door_open and motion)
            if rng.random() < 0.02:
                next_state ^= 1
            records.append(
                {
                    "home": home_id,
                    "step": step,
                    "features": [state, door_open, motion],
                    "label": next_state,
                }
            )
            state = next_state
    return homes, records


def digest_records(records):
    payload = json.dumps(records, separators=(",", ":"), sort_keys=True).encode()
    return hashlib.sha256(payload).hexdigest()


def sigmoid(value):
    if value >= 0:
        z = math.exp(-min(value, 40.0))
        return 1.0 / (1.0 + z)
    z = math.exp(max(value, -40.0))
    return z / (1.0 + z)


def new_model(seed):
    rng = random.Random(seed)
    scale = math.sqrt(2.0 / 3.0)
    return {
        "w1": [[rng.uniform(-scale, scale) for _ in range(3)] for _ in range(HIDDEN_SIZE)],
        "b1": [0.0 for _ in range(HIDDEN_SIZE)],
        "w2": [rng.uniform(-0.25, 0.25) for _ in range(HIDDEN_SIZE)],
        "b2": 0.0,
    }


def predict_probability(model, features):
    hidden = []
    for weights, bias in zip(model["w1"], model["b1"]):
        activation = bias + sum(weight * value for weight, value in zip(weights, features))
        hidden.append(math.tanh(activation))
    logit = model["b2"] + sum(weight * value for weight, value in zip(model["w2"], hidden))
    return sigmoid(logit)


def train_batch(model, batch):
    grad_w1 = [[0.0] * 3 for _ in range(HIDDEN_SIZE)]
    grad_b1 = [0.0] * HIDDEN_SIZE
    grad_w2 = [0.0] * HIDDEN_SIZE
    grad_b2 = 0.0
    for record in batch:
        features = record["features"]
        hidden = []
        for weights, bias in zip(model["w1"], model["b1"]):
            hidden.append(math.tanh(bias + sum(w * x for w, x in zip(weights, features))))
        probability = sigmoid(model["b2"] + sum(w * h for w, h in zip(model["w2"], hidden)))
        output_delta = probability - record["label"]
        grad_b2 += output_delta
        for unit in range(HIDDEN_SIZE):
            grad_w2[unit] += output_delta * hidden[unit]
            hidden_delta = output_delta * model["w2"][unit] * (1.0 - hidden[unit] ** 2)
            grad_b1[unit] += hidden_delta
            for feature_index, value in enumerate(features):
                grad_w1[unit][feature_index] += hidden_delta * value

    inverse_batch = 1.0 / len(batch)
    for unit in range(HIDDEN_SIZE):
        for feature_index in range(3):
            weight = model["w1"][unit][feature_index]
            gradient = grad_w1[unit][feature_index] * inverse_batch + L2 * weight
            model["w1"][unit][feature_index] -= LEARNING_RATE * gradient
        model["b1"][unit] -= LEARNING_RATE * grad_b1[unit] * inverse_batch
        model["w2"][unit] -= LEARNING_RATE * (grad_w2[unit] * inverse_batch + L2 * model["w2"][unit])
    model["b2"] -= LEARNING_RATE * grad_b2 * inverse_batch


def log_loss(records, model):
    total = 0.0
    for record in records:
        p = min(max(predict_probability(model, record["features"]), 1e-12), 1 - 1e-12)
        y = record["label"]
        total -= y * math.log(p) + (1 - y) * math.log(1 - p)
    return total / len(records)


def fit(train, validation):
    model = new_model(MODEL_SEED)
    rng = random.Random(MODEL_SEED + 1)
    best_model = None
    best_validation_loss = float("inf")
    best_epoch = 0
    epochs_without_improvement = 0
    for epoch in range(1, MAX_EPOCHS + 1):
        order = list(range(len(train)))
        rng.shuffle(order)
        for start in range(0, len(order), BATCH_SIZE):
            batch = [train[index] for index in order[start : start + BATCH_SIZE]]
            train_batch(model, batch)
        current_loss = log_loss(validation, model)
        if current_loss < best_validation_loss - 1e-8:
            best_validation_loss = current_loss
            best_model = json.loads(json.dumps(model))
            best_epoch = epoch
            epochs_without_improvement = 0
        else:
            epochs_without_improvement += 1
            if epochs_without_improvement >= PATIENCE:
                break
    return best_model, best_epoch, best_validation_loss


def metrics(records, predict):
    tp = tn = fp = fn = 0
    brier = 0.0
    loss = 0.0
    for record in records:
        probability = min(max(predict(record), 1e-12), 1 - 1e-12)
        predicted = int(probability >= 0.5)
        actual = record["label"]
        tp += int(predicted == 1 and actual == 1)
        tn += int(predicted == 0 and actual == 0)
        fp += int(predicted == 1 and actual == 0)
        fn += int(predicted == 0 and actual == 1)
        brier += (probability - actual) ** 2
        loss -= actual * math.log(probability) + (1 - actual) * math.log(1 - probability)
    total = len(records)
    accuracy = (tp + tn) / total
    f1_positive = 2 * tp / max(1, 2 * tp + fp + fn)
    f1_negative = 2 * tn / max(1, 2 * tn + fp + fn)
    return {
        "accuracy": round(accuracy, 6),
        "macro_f1": round((f1_positive + f1_negative) / 2, 6),
        "brier_score": round(brier / total, 6),
        "log_loss": round(loss / total, 6),
        "confusion": {"true_negative": tn, "false_positive": fp, "false_negative": fn, "true_positive": tp},
        "count": total,
    }


def main():
    homes, records = generate_data()
    train_homes = set(homes[:TRAIN_HOMES])
    validation_end = TRAIN_HOMES + VALIDATION_HOMES
    validation_homes = set(homes[TRAIN_HOMES:validation_end])
    test_homes = set(homes[validation_end:])
    train = [record for record in records if record["home"] in train_homes]
    validation = [record for record in records if record["home"] in validation_homes]
    test = [record for record in records if record["home"] in test_homes]
    assert train_homes.isdisjoint(validation_homes | test_homes)
    assert validation_homes.isdisjoint(test_homes)
    model, best_epoch, best_validation_loss = fit(train, validation)
    majority = int(sum(record["label"] for record in train) * 2 >= len(train))
    learned = metrics(test, lambda record: predict_probability(model, record["features"]))
    majority_result = metrics(test, lambda _: float(majority))
    last_state = metrics(test, lambda record: float(record["features"][0]))
    model_bytes = json.dumps(model, separators=(",", ":"), sort_keys=True).encode()
    output = {
        "experiment": "synora-i1-next-state-mlp/v1",
        "status": "EXPERIMENTAL",
        "data_seed": DATA_SEED,
        "model_seed": MODEL_SEED,
        "dataset_sha256": digest_records(records),
        "split": {
            "method": "whole-home holdout; no home occurs in multiple splits",
            "home_counts": {"train": len(train_homes), "validation": len(validation_homes), "test": len(test_homes)},
            "record_counts": {"train": len(train), "validation": len(validation), "test": len(test)},
            "home_ids": {"train": sorted(train_homes), "validation": sorted(validation_homes), "test": sorted(test_homes)},
        },
        "synthetic_world": {
            "features": ["current_occupancy_bit", "door_open_bit", "motion_bit"],
            "label": "next_occupancy_bit",
            "transition": "occupancy toggles when door_open AND motion; independent 2% label noise",
            "home_variation": "per-home door and motion frequencies vary; transition rule is shared",
            "missing_data": "excluded from this first experiment; no missing value is coerced to false",
        },
        "model": {
            "architecture": "3-input, 6-unit tanh hidden layer, 1 sigmoid output, binary cross entropy",
            "parameter_count": 31,
            "optimizer": "deterministic mini-batch SGD with L2; validation-loss early stopping",
            "batch_size": BATCH_SIZE,
            "learning_rate": LEARNING_RATE,
            "l2": L2,
            "max_epochs": MAX_EPOCHS,
            "best_epoch": best_epoch,
            "best_validation_log_loss": round(best_validation_loss, 6),
            "model_sha256": hashlib.sha256(model_bytes).hexdigest(),
        },
        "test_metrics": {
            "learned_mlp": learned,
            "majority_class": majority_result,
            "last_state_persistence": last_state,
            "accuracy_delta_vs_majority": round(learned["accuracy"] - majority_result["accuracy"], 6),
            "accuracy_delta_vs_last_state": round(learned["accuracy"] - last_state["accuracy"], 6),
        },
        "qualification": "synthetic proof of concept only; not household generalization, production model, or deployment evidence",
    }
    print(json.dumps(output, indent=2, sort_keys=True))


if __name__ == "__main__":
    main()
