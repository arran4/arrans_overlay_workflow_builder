with open('readme.md', 'r') as f:
    content = f.read()

# Find the Note on TAG vs tag and remove it
content = content.replace("""**Note on `${TAG}` vs `${tag}`:**
It is highly recommended to use `${TAG}` (or `${VERSION}`) in your configuration file. The generator will safely convert it into the correct loop variable declaration (`${tag}`) within the bash script used by the GitHub actions workflow. If you use `${tag}` directly, it will still work but using the capitalized version explicitly signals to the generator that this is a placeholder meant to be substituted.

""", "")

with open('readme.md', 'w') as f:
    f.write(content)
