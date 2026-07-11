import sys

def replace_in_file(filepath, old_text, new_text):
    with open(filepath, 'r') as f:
        content = f.read()
    if old_text in content:
        content = content.replace(old_text, new_text)
        with open(filepath, 'w') as f:
            f.write(content)
        print(f"Replaced in {filepath}")
    else:
        print(f"Old text not found in {filepath}")

replace_in_file('templates/github-binary.tmpl',
    'echo "$version == $tag so there is no [[- .WorkaroundTagPrefix ]] removed skipping"',
    'echo "$version == $tag so there is no [[ .WorkaroundTagPrefix ]] removed skipping"')
