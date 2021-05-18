#!/usr/bin/env python3
# @author: marcel.fest<at>live.de
# @requires pyyaml
## Example:
# .bin/rename.py --pattern '^(.*?)(-configmap)(.*?\.yaml)$' --substitution '\1\3'
#
import glob
import argparse
import re
import os
import shutil
import pathlib

parser = argparse.ArgumentParser(description='Rename some files.')
parser.add_argument('--pattern', type=str, default='')
parser.add_argument('--substitution', type=str, default='')
parser.add_argument('--directory', dest='directory', action='store_true')
parser.set_defaults(directory=False)
args = parser.parse_args()

pattern = re.compile(args.pattern)
files = glob.glob('**', recursive=True)
if args.directory:
    files = [filename for filename in files if pathlib.Path(filename).is_dir()]
files = [filename for filename in files if pattern.match(str(filename))]


for filename in files:
    new_filename = pattern.sub(args.substitution, filename)
    print(filename, '->', new_filename)
    if args.directory:
        shutil.move(filename, new_filename)
    else:
        os.rename(filename, new_filename)
# ^(.*?)(-configmap)(.*?\.yaml)$/\1\3
# '^(.*?/)prod$' --substitution='\1prd'
# .bin/rename.py --directory --pattern='^(.*?/)prod$' --substitution='\1prd'