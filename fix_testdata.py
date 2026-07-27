import re

with open('test_output.txt', 'r') as f:
    text = f.read()

parts = text.split("Result was:\n")

for filepath in ['testdata/txtar/web-binary/web-binary.txtar', 'testdata/txtar/github-cmake/kllamabooks.txtar']:
    with open(filepath, 'r') as f:
        content = f.read()

    for part in parts[1:]:
        if ('google-cloud-sdk-bin update' in part and 'web-binary' in filepath) or ('kjules' in part and 'kllamabooks' in filepath):
            end_idx = part.find('--- FAIL:')
            if end_idx != -1:
                yaml_content = part[:end_idx].strip()
            else:
                yaml_content = part.strip()

            yaml_content = "\n".join([line[8:] if line.startswith("        ") else line for line in yaml_content.split("\n")])

            split_idx = content.find("-- expected.yaml --")
            if split_idx != -1:
                new_content = content[:split_idx + len("-- expected.yaml --\n")] + yaml_content + "\n"

                new_content = re.sub(r'\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}(?:\.\d+)? \+\d{4} UTC(?: m=\+\d+\.\d+)?', '2006-01-02 15:04:05.999999999 -0700 MST', new_content)
                new_content = re.sub(r'1\.0\.0 Web Binary test.config', '1.0.0 Web Binary testdata/txtar/web-binary/web-binary.txtar/input.config', new_content)
                new_content = re.sub(r'1\.0\.0 Github Cmake Release test.config', '1.0.0 Github Cmake Release testdata/txtar/github-cmake/kllamabooks.txtar/input.config', new_content)

                with open(filepath, 'w') as f:
                    f.write(new_content)
            break
