import sys
import urllib.request
import re
import json
import xml.etree.ElementTree as ET
from urllib.parse import urlparse

def execute_pipeline(pipeline_str):
    commands = [cmd.strip() for cmd in pipeline_str.split('|')]
    data = None

    for cmd in commands:
        if cmd.startswith('get('):
            url = cmd[4:-1]
            req = urllib.request.Request(url, headers={'User-Agent': 'Mozilla/5.0'})
            with urllib.request.urlopen(req) as response:
                data = response.read().decode('utf-8')
        elif cmd in ('rss', 'atom'):
            try:
                root = ET.fromstring(data)
                items = root.findall('.//item') + root.findall('.//{http://www.w3.org/2005/Atom}entry')
                data = items
            except Exception:
                data = []
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
            obj = json.loads(data)
            for part in path.split('.'):
                if part and isinstance(obj, dict):
                    obj = obj.get(part)
                elif part.isdigit() and isinstance(obj, list):
                    obj = obj[int(part)]
            data = obj
        elif cmd.startswith('replace('):
            args = cmd[8:-1].split(',')
            if len(args) >= 2 and data and isinstance(data, str):
                data = data.replace(args[0].strip("'\""), args[1].strip("'\""))
        elif cmd == 'trim':
             if isinstance(data, str):
                  data = data.strip()
        elif cmd == 'html_links':
             if isinstance(data, str):
                 data = re.findall(r'href=[\'"]?([^\'" >]+)', data)

    if isinstance(data, list):
        for item in data:
            if item: print(item)
    elif data:
        print(data)

if __name__ == '__main__':
    if len(sys.argv) > 1:
        execute_pipeline(sys.argv[1])
