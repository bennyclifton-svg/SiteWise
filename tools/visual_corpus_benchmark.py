"""Sequential, read-only benchmark of exported production filing requests.

First use cmd/titleblock-experiment -visual to export source evidence. Keep
exports, results and manually reviewed answer keys in a private directory.
This script never reads an answer key, writes application data, or parallelizes
provider calls. https://docs.typesafe.ai/images
"""
import argparse
import hashlib
import json
from pathlib import Path

from titleblock_experiment import ask


def run(manifest, exports, output, mode):
    items = json.loads(manifest.read_text(encoding="utf-8-sig"))
    if not 1 <= len(items) <= 40:
        raise ValueError("select 1-40 documents")
    output.mkdir(parents=True, exist_ok=True)
    summaries = []
    for item in items:
        identifier = item["id"]
        if Path(identifier).name != identifier or identifier in ("", ".", ".."):
            raise ValueError("invalid sample id")
        evidence = json.loads((exports / (identifier + ".json")).read_text(encoding="utf-8"))
        if evidence["sha256"] != item["sha256"]:
            raise ValueError("source hash mismatch")
        data = evidence[mode]
        summary = {"id": identifier, "sha256": item["sha256"], "mode": mode}
        try:
            if "error" in data:
                raise ValueError(data["error"])
            request = data["request"]
            target = output / (identifier + "-" + mode + ".json")
            fingerprint = hashlib.sha256(json.dumps(request).encode()).hexdigest()
            if target.exists() and json.loads(target.read_text())["request_sha256"] != fingerprint:
                raise ValueError("cached request differs; choose a new output directory")
            result = ask(request, target) if request.get("questions") else {"response": {"answers": {}}, "ms": 0}
            picked = {d["Field"]: d["Value"] for d in data.get("decisions", [])}
            for field, answer in result["response"]["answers"].items():
                if answer.get("type") != "choice":
                    continue
                choice = answer.get("choice", "none")
                candidate = request["questions"][field].get("criteria", {}).get(choice)
                picked[field] = candidate.get("value") if isinstance(candidate, dict) else None
            prep = data.get("prepare_ms", data.get("extract_ms", 0) + data.get("harvest_ms", 0))
            summary.update(values=picked, prepare_ms=prep, api_ms=result["ms"], total_ms=prep + result["ms"])
        except Exception as error:
            summary["error"] = str(error)
        summaries.append(summary)
        (output / "summary.json").write_text(json.dumps(summaries, indent=2), encoding="utf-8")
        print(identifier, summary.get("values", {}).get("title"), summary.get("error", ""), flush=True)


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--manifest", type=Path, required=True)
    parser.add_argument("--exports", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--mode", choices=["visual", "native", "ocr"], default="visual")
    args = parser.parse_args()
    run(args.manifest, args.exports, args.output, args.mode)
