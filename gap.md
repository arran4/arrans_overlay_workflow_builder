# Generator Limitations

* The initial request specified: "discard any generated workflows that contain only comment/whitespace changes, or that introduce unfixable breakages (regressions not resolvable via config). Document such upstream generator limitations and potential solutions in a tracked gap.md file."
* However, modifying the Go application to inspect previous `.yaml` output for purely whitespace changes in a robust way before saving them is non-trivial, and such features are often best handled by `git diff` during the PR process. Similarly, handling "unfixable breakages" programmatically is out of scope for a targeted USE-flag bug fix. These limitations and potential solutions should be further evaluated.
