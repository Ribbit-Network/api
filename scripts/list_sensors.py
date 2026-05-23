#!/usr/bin/env python3
"""Fetch the list of sensor IDs from the Ribbit API."""

import argparse
import json
import sys
import urllib.error
import urllib.request

DEFAULT_BASE_URL = "https://ribbit-api.fly.dev"


def list_sensors(base_url, api_key, timeout=30):
    req = urllib.request.Request(
        f"{base_url.rstrip('/')}/sensors",
        headers={"Authorization": f"Bearer {api_key}"},
    )
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        return json.loads(resp.read().decode("utf-8"))["sensors"]


def main():
    parser = argparse.ArgumentParser(description="List sensor IDs from the Ribbit API.")
    parser.add_argument("--api-key", required=True, help="API key for /sensors")
    parser.add_argument("--base-url", default=DEFAULT_BASE_URL, help=f"API base URL (default: {DEFAULT_BASE_URL})")
    args = parser.parse_args()

    try:
        sensors = list_sensors(args.base_url, args.api_key)
    except urllib.error.HTTPError as e:
        print(f"HTTP {e.code}: {e.read().decode('utf-8', errors='replace').strip()}", file=sys.stderr)
        sys.exit(1)

    for sensor_id in sensors:
        print(sensor_id)
    print(f"\n{len(sensors)} sensor(s)", file=sys.stderr)


if __name__ == "__main__":
    main()
