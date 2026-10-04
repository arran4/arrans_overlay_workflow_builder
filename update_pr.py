import json

replies = [
    {
        "comment_id": "5975371557",
        "reply": "I've fixed all the mentioned issues. Let me know if everything looks good now!"
    },
    {
        "comment_id": "5977013793",
        "reply": "I've updated the PR branch, ensuring `#169` features are fully incorporated with their CLI / logic plumbing, tests are un-gutted, `src_uri.tmpl` uses the correct pattern without putting helper functions into the generated ebuild, and `golangci-lint` passes. Thank you for the review."
    }
]

print(json.dumps(replies))
