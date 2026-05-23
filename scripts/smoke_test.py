#!/usr/bin/env python3
"""Smoke test for the production Ribbit API."""

import argparse
import json
import sys
import urllib.error
import urllib.parse
import urllib.request
from datetime import datetime, timedelta, timezone

DEFAULT_BASE_URL = "https://ribbit-api.fly.dev"


def fetch(url, headers=None, timeout=30):
    req = urllib.request.Request(url, headers=headers or {})
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            return resp.status, resp.read().decode("utf-8")
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode("utf-8", errors="replace")


def check(name, ok, detail=""):
    mark = "PASS" if ok else "FAIL"
    print(f"[{mark}] {name}")
    if detail:
        for line in detail.splitlines():
            print(f"       {line}")
    return ok


def main():
    parser = argparse.ArgumentParser(description="Smoke-test the Ribbit API.")
    parser.add_argument("--api-key", required=True, help="API key for /data")
    parser.add_argument("--base-url", default=DEFAULT_BASE_URL, help=f"API base URL (default: {DEFAULT_BASE_URL})")
    parser.add_argument("--hours", type=int, default=24, help="Look-back window for /data (default: 24)")
    args = parser.parse_args()

    base = args.base_url.rstrip("/")
    auth_headers = {"Authorization": f"Bearer {args.api_key}"}
    results = []

    status, body = fetch(f"{base}/")
    results.append(check(
        "GET /  → 200 with frog",
        status == 200 and "🐸" in body,
        f"status={status} body={body.strip()!r}",
    ))

    status, body = fetch(f"{base}/healthz")
    results.append(check(
        "GET /healthz  → 200 ok",
        status == 200 and "ok" in body,
        f"status={status} body={body.strip()!r}",
    ))

    status, body = fetch(f"{base}/data?start=2024-01-01T00:00:00Z")
    results.append(check(
        "GET /data without key  → 401",
        status == 401,
        f"status={status} body={body.strip()!r}",
    ))

    start = (datetime.now(timezone.utc) - timedelta(hours=args.hours)).strftime("%Y-%m-%dT%H:%M:%SZ")
    query = urllib.parse.urlencode({"start": start, "fields": "co2,lat,lon", "interval": "1h"})
    status, body = fetch(f"{base}/data?{query}", headers=auth_headers)

    parsed_ok, rows, parse_detail = False, 0, ""
    if status == 200:
        try:
            payload = json.loads(body)
            data = payload.get("data", [])
            parsed_ok = isinstance(data, list)
            rows = len(data)
            sample = json.dumps(data[0], indent=2) if data else "(empty data array)"
            parse_detail = f"rows={rows}\nsample={sample}"
        except json.JSONDecodeError as e:
            parse_detail = f"json decode error: {e}\nbody={body[:300]!r}"
    else:
        parse_detail = f"status={status} body={body[:500]!r}"

    results.append(check(
        f"GET /data (last {args.hours}h, co2/lat/lon, 1h)  → 200 JSON",
        status == 200 and parsed_ok,
        parse_detail,
    ))

    results.append(check(
        "  ↳ returned at least one row",
        status == 200 and rows > 0,
        f"rows={rows}",
    ))

    passed = sum(1 for r in results if r)
    print(f"\n{passed}/{len(results)} checks passed")
    sys.exit(0 if passed == len(results) else 1)


if __name__ == "__main__":
    main()
