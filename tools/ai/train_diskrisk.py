#!/usr/bin/env python3
"""Train MoodNAS's disk-failure risk model from Backblaze drive-stats data.

Backblaze publishes daily SMART snapshots of its fleet with a `failure`
column (https://www.backblaze.com/cloud-storage/resources/hard-drive-test-data).
Download one or more quarters, unzip them, and run:

    python3 -I tools/ai/train_diskrisk.py data_Q1_2025/ data_Q2_2025/ -o diskrisk-model.json

Copy the resulting JSON to /var/lib/moodnas/diskrisk-model.json on the NAS
(or the -data directory); moodd loads it at start-up instead of the built-in
priors. Feature definitions must match internal/ai/local/diskrisk.go.

Labelling: a drive-day is positive if that drive fails within HORIZON days
after it. Negatives are subsampled. Plain logistic regression trained with
batch gradient descent and L2; standard library only.
"""
import argparse
import csv
import datetime as dt
import json
import math
import random
import sys
from pathlib import Path

HORIZON = 30
FEATURES = ["reallocated", "reported_uncorrectable", "command_timeout", "pending",
            "offline_uncorrectable", "age_years", "temp_over_45"]


def num(row, col):
    v = row.get(col) or ""
    try:
        return float(v)
    except ValueError:
        return 0.0


def featurize(row):
    l1p = lambda v: math.log1p(max(v, 0.0))
    return [
        l1p(num(row, "smart_5_raw")),
        l1p(num(row, "smart_187_raw")),
        l1p(float(int(num(row, "smart_188_raw")) & 0xFFFF)),
        l1p(num(row, "smart_197_raw")),
        l1p(num(row, "smart_198_raw")),
        num(row, "smart_9_raw") / 8766.0,
        max(0.0, num(row, "smart_194_raw") - 45.0),
    ]


def load(dirs, neg_rate, rng):
    files = sorted(p for d in dirs for p in Path(d).glob("*.csv"))
    if not files:
        sys.exit("no CSV files found")
    fail_date = {}
    for f in files:  # pass 1: when does each drive fail?
        with f.open(newline="") as fh:
            for row in csv.DictReader(fh):
                if row.get("failure") == "1":
                    fail_date[row["serial_number"]] = dt.date.fromisoformat(row["date"])
    X, y = [], []
    for f in files:  # pass 2: label drive-days
        with f.open(newline="") as fh:
            for row in csv.DictReader(fh):
                fd = fail_date.get(row["serial_number"])
                day = dt.date.fromisoformat(row["date"])
                pos = fd is not None and 0 <= (fd - day).days <= HORIZON
                if pos or rng.random() < neg_rate:
                    X.append(featurize(row))
                    y.append(1.0 if pos else 0.0)
    return X, y


def train(X, y, epochs, lr, l2, pos_weight):
    n, k = len(X), len(FEATURES)
    w, b = [0.0] * k, -3.0
    for ep in range(epochs):
        gw, gb, loss = [0.0] * k, 0.0, 0.0
        for xi, yi in zip(X, y):
            z = b + sum(wj * xj for wj, xj in zip(w, xi))
            p = 1 / (1 + math.exp(-max(min(z, 30), -30)))
            s = pos_weight if yi else 1.0
            err = (p - yi) * s
            for j in range(k):
                gw[j] += err * xi[j]
            gb += err
            loss -= s * (yi * math.log(p + 1e-12) + (1 - yi) * math.log(1 - p + 1e-12))
        for j in range(k):
            w[j] -= lr * (gw[j] / n + l2 * w[j])
        b -= lr * gb / n
        if ep % 50 == 0:
            print(f"epoch {ep:4d}  loss {loss / n:.5f}", file=sys.stderr)
    return w, b


def main():
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument("dirs", nargs="+", help="unzipped Backblaze data folders")
    ap.add_argument("-o", "--output", default="diskrisk-model.json")
    ap.add_argument("--neg-rate", type=float, default=0.002, help="fraction of healthy drive-days kept")
    ap.add_argument("--epochs", type=int, default=400)
    ap.add_argument("--lr", type=float, default=0.5)
    ap.add_argument("--l2", type=float, default=1e-3)
    ap.add_argument("--seed", type=int, default=7)
    a = ap.parse_args()
    rng = random.Random(a.seed)
    X, y = load(a.dirs, a.neg_rate, rng)
    pos = sum(y)
    print(f"{len(X)} samples, {int(pos)} positive", file=sys.stderr)
    if pos == 0:
        sys.exit("no failures in the data: use more quarters")
    # undo the negative subsampling so probabilities stay calibrated-ish,
    # then balance classes during optimisation
    w, b = train(X, y, a.epochs, a.lr, a.l2, pos_weight=(len(y) - pos) / pos)
    b += math.log(a.neg_rate) - math.log((len(y) - pos) / pos)
    model = {
        "bias": round(b, 4),
        "weights": {f: round(v, 4) for f, v in zip(FEATURES, w)} | {"smart_failed": 3.0},
        "source": f"Backblaze drive stats, {len(a.dirs)} folder(s), horizon {HORIZON} d, {int(pos)} failures",
    }
    Path(a.output).write_text(json.dumps(model, indent=2) + "\n")
    print(f"wrote {a.output}", file=sys.stderr)


if __name__ == "__main__":
    main()
