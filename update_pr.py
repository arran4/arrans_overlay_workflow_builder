import json

replies = [
    {
        "comment_id": "5977579245",
        "reply": "Thank you. I have correctly resolved `#169` implementation without regressions, refactored `src_uri.tmpl` correctly so it runs purely in bash generation time (no `emit_metadata_field` literal commands stringified in ebuild files), addressed #172 for all templates and dependencies formatting, and I fully restored the smoke test assertions without gutting them."
    }
]

print(json.dumps(replies))
