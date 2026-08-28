import sys
import urllib.request
import re
import json
import xml.etree.ElementTree as ET
from urllib.parse import urlparse, urljoin

def tokenize_pipeline(pipeline_str):
    commands = []
    current_cmd = ""
    paren_depth = 0
    in_quote = None
    escape_next = False

    for char in pipeline_str:
        if escape_next:
            current_cmd += char
            escape_next = False
            continue

        if char == '\\':
            escape_next = True
            current_cmd += char
            continue

        if char in ("'", '"'):
            if in_quote == char:
                in_quote = None
            elif not in_quote:
                in_quote = char
            current_cmd += char
            continue

        if not in_quote:
            if char == '(':
                paren_depth += 1
            elif char == ')':
                paren_depth -= 1
                if paren_depth < 0:
                    print("Error: Unbalanced parentheses in pipeline", file=sys.stderr)
                    sys.exit(1)
            elif char == '|' and paren_depth == 0:
                cmd = current_cmd.strip()
                if not cmd:
                    print("Error: Empty pipeline stage", file=sys.stderr)
                    sys.exit(1)
                commands.append(cmd)
                current_cmd = ""
                continue

        current_cmd += char

    if in_quote:
        print("Error: Unterminated quote in pipeline", file=sys.stderr)
        sys.exit(1)
    if paren_depth > 0:
        print("Error: Unbalanced parentheses in pipeline", file=sys.stderr)
        sys.exit(1)

    cmd = current_cmd.strip()
    if not cmd:
        print("Error: Empty pipeline stage", file=sys.stderr)
        sys.exit(1)
    commands.append(cmd)

    return commands

def execute_pipeline(pipeline_str):
    commands = tokenize_pipeline(pipeline_str)
    data = None
    current_url = ""

    for cmd in commands:
        if cmd.startswith('get('):
            url = cmd[4:-1].strip("'\"")
            current_url = url
            req = urllib.request.Request(url, headers={'User-Agent': 'Mozilla/5.0'})
            try:
                with urllib.request.urlopen(req) as response:
                    data = response.read().decode('utf-8')
            except Exception as e:
                print(f"Error fetching URL {url}: {e}", file=sys.stderr)
                sys.exit(1)
        elif cmd in ('rss', 'atom'):
            try:
                root = ET.fromstring(data)
                items = root.findall('.//item') + root.findall('.//{http://www.w3.org/2005/Atom}entry')
                data = items
            except Exception as e:
                print(f"Error parsing XML/RSS: {e}", file=sys.stderr)
                sys.exit(1)
        elif cmd == 'xml':
            try:
                data = ET.fromstring(data)
            except Exception as e:
                print(f"Error parsing XML: {e}", file=sys.stderr)
                sys.exit(1)
        elif cmd.startswith('xpath('):
            xpath_query = cmd[6:-1].strip("'\"")
            try:
                root = ET.fromstring(data) if isinstance(data, str) else data
                results = root.findall(xpath_query)
                data = [r.text if r.text else r.get('href', '') for r in results]
            except Exception as e:
                print(f"Error executing xpath: {e}", file=sys.stderr)
                sys.exit(1)
        elif cmd == 'first':
            data = data[0] if data and isinstance(data, list) else data
        elif cmd == 'last':
            data = data[-1] if data and isinstance(data, list) else data
        elif cmd == 'link':
            if data is not None:
                link = data.find('link')
                if link is not None:
                    if link.text and link.text.strip():
                        data = link.text.strip()
                    else:
                        data = link.get('href', '')
                else:
                    link = data.find('{http://www.w3.org/2005/Atom}link')
                    if link is not None:
                        data = link.get('href', '')
                    else:
                        data = ''
        elif cmd == 'url.basename':
            if data and isinstance(data, str):
                path = urlparse(data).path
                data = path.split('/')[-1] if path else ''
        elif cmd.startswith('regex('):
            pattern = cmd[6:-1]
            if isinstance(data, list):
                res = []
                for item in data:
                    item_str = ET.tostring(item, encoding='unicode') if isinstance(item, ET.Element) else str(item)
                    m = re.search(pattern, item_str)
                    if m:
                        res.append(m.group(1) if m.groups() else m.group(0))
                data = res
            elif data:
                data_str = ET.tostring(data, encoding='unicode') if isinstance(data, ET.Element) else str(data)
                m = re.search(pattern, data_str)
                if m:
                    data = m.group(1) if m.groups() else m.group(0)
                else:
                    data = ""
        elif cmd.startswith('json('):
            path = cmd[5:-1]
            try:
                obj = json.loads(data)
                for part in path.split('.'):
                    if part and isinstance(obj, dict):
                        obj = obj.get(part)
                    elif part.isdigit() and isinstance(obj, list):
                        obj = obj[int(part)]
                data = obj
            except Exception as e:
                print(f"Error parsing JSON path {path}: {e}", file=sys.stderr)
                sys.exit(1)
        elif cmd.startswith("replace(") and cmd.endswith(")"):
            args_str = cmd[len("replace("):-1]
            try:
                import ast
                args = ast.literal_eval(f"({args_str},)")
                if not isinstance(args, tuple) or len(args) != 2:
                    raise ValueError("replace requires exactly 2 arguments")
                if not all(isinstance(arg, str) for arg in args):
                    raise ValueError("replace arguments must be strings")
                old, new = args
            except (SyntaxError, ValueError) as e:
                print(f"Error parsing replace arguments: {e}", file=sys.stderr)
                sys.exit(1)

            if isinstance(data, str):
                data = data.replace(old, new)
        elif cmd == 'trim':
             if isinstance(data, str):
                  data = data.strip()
        elif cmd == 'html_links':
             from html.parser import HTMLParser
             class LinkParser(HTMLParser):
                 def __init__(self):
                     super().__init__()
                     self.links = []
                 def handle_starttag(self, tag, attrs):
                     if tag == 'a':
                         for attr in attrs:
                             if attr[0] == 'href':
                                 self.links.append(attr[1])

             if isinstance(data, str):
                 parser = LinkParser()
                 parser.feed(data)
                 links = parser.links
                 if current_url:
                     links = [urljoin(current_url, link) for link in links]
                 data = links
        else:
             print(f"Unknown command: {cmd}", file=sys.stderr)
             sys.exit(1)

    if isinstance(data, list):
        for item in data:
            if item: print(item)
    elif data:
        print(data)

if __name__ == '__main__':
    if len(sys.argv) > 1:
        execute_pipeline(sys.argv[1])
