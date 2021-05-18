#!/usr/bin/env python3
# @author: marcel.fest<at>live.de
# @requires pyyaml
import yaml
import json
import glob
import os
import argparse
from pathlib import Path
from typing import Tuple

yaml.preserve_quotes = True

resourceKindNames = {"customresourcedefinition": "crds", "networkpolicy": "networkpolicies", "configmap": "configmaps",
                     "secret": "secrets", "namespace": "namespaces", "clusterrole": "clusterroles", "service": "services",
                     "gitrepository": "gitrepositories", "kustomization": "kustomizations", "helmrelease": "helmreleases",
                      "clusterrolebinding": "clusterrolebindings", "serviceaccount": "serviceaccounts"}


class bcolors:
    HEADER = '\033[95m'
    OKBLUE = '\033[94m'
    OKCYAN = '\033[96m'
    OKGREEN = '\033[92m'
    WARNING = '\033[93m'
    FAIL = '\033[91m'
    ENDC = '\033[0m'
    BOLD = '\033[1m'
    UNDERLINE = '\033[4m'


def get_name(filename: str, ending: str = 'yaml') -> str:
    name, _ = filename.split(f'.{ending}', maxsplit=1)
    return name


def get_len_error_message(data, name_len):
    return f"{bcolors.FAIL}{data['metadata']['name']} length is {name_len} this is longer then the maximum of 63{bcolors.ENDC}"


def get_error_message(path, filename, composed_name):
    return f"""{bcolors.FAIL}
The file and the name are not the same:
    PATH: {path}
    FILE: {filename}
    NAME: {composed_name}
{bcolors.ENDC}"""


def get_crd_folder_error_message(path, filename, composed_name, top_folder):
    return f"""{bcolors.FAIL}
The  file is not placed in the correct folder:
    PATH: {path}
    TOP FOLDER: {top_folder}
{bcolors.ENDC}"""


def check_name(data: dict, filename_without_path: str, top_folder: str = "", exception: bool = False, show_errors: bool = False, filename: str = "") -> Tuple[bool, str, str]:
    error = False
    isCRD = False
    resourceKind = None
    try:
        composed_name = f"{data['metadata']['name']}_{data['metadata']['namespace']}"
    except KeyError:
        try: 
            composed_name = data['metadata']['name']
        except KeyError:
            composed_name = filename_without_path
    except TypeError:
        return False, "", False

    try:
        resourceKind = resourceKindNames[data['kind'].lower()]
    except KeyError:
        pass
    try:
        assert filename_without_path == composed_name, \
            get_error_message(filename, filename_without_path, composed_name)
    except AssertionError as e1:
        if exception:
            raise e1
        print(e1)
        error = True
    try:
        if resourceKind and resourceKind not in top_folder:
            raise Exception(get_crd_folder_error_message(
                filename, filename_without_path, composed_name, top_folder))
    except UnboundLocalError:
        pass
    except Exception as e1:
        if exception:
            raise e1
        print(e1)
        error = True
    try:
        if data and (name_len := len(data['metadata']['name']) > 63):
            raise Exception(get_len_error_message(data, name_len))
        else:
            if not error and not show_errors:
                print(
                    f'{bcolors.OKGREEN}{filename} [{len(data["metadata"]["name"] if data else [])}]{bcolors.ENDC}')
    except KeyError:
        pass
    return error, composed_name, resourceKind


def rename_file(data: dict, filename: str, composed_name: str, resourceKind: str = None, single: bool = True):
    print("Try to auto fix the filenaming")
    path = f"{os.sep}".join(filename.split(f'{os.sep}')[:-1])
    current_folder = path.split(f'{os.sep}')[-1]
    if resourceKind and resourceKind != current_folder:
        new_folder = f"{path}{os.sep}{resourceKind}"
        new_filename = f"{new_folder}{os.sep}{composed_name}.yaml"
        if not os.path.exists(new_folder):
            os.makedirs(new_folder)
    else:
        new_filename = f"{path}{os.sep}{composed_name}.yaml"
    print(filename, '->', new_filename)
    if not single:
        with open(new_filename, 'w') as new_file:
            yaml.dump(data, new_file, Dumper=yaml.SafeDumper)
    else:
        os.rename(filename, new_filename)


def validate(files, exception: bool = False, show_errors: bool = False, rename: bool = False):
    for filename in files:
        splitted_path = filename.split(f'{os.sep}')
        filename_without_path = splitted_path[-1]
        try:
            top_folder = splitted_path[-2]
        except IndexError:
            top_folder = ""
        with open(filename, 'r') as file:
            datas = [{}]
            if filename_without_path.endswith('.yaml'):
                filename_without_path = get_name(filename_without_path)
                datas = yaml.load_all(file, Loader=yaml.SafeLoader)
            elif filename.endswith('.json'):
                filename_without_path = get_name(
                    filename_without_path, ending='json')
                datas = json.load(file)
                if not isinstance(data, list):
                    datas = [data]
            else:
                datas = [{
                    'metadata': {
                        'name': filename_without_path
                    }
                }]
            # This is only in place to resolve the generator
            # for yaml load_all
            datas = list(datas)
            for data in datas:
                error, composed_name, resourceKind = check_name(
                    data,
                    filename_without_path,
                    exception=exception,
                    show_errors=show_errors,
                    filename=filename,
                    top_folder=top_folder
                )
                if error and rename:
                    single = True
                    if len(datas) > 1:
                        single = False
                    elif len(datas) == 1:
                        single = True
                    elif len(datas) < 1:
                        raise NotImplementedError("This should never happen.")

                    rename_file(data, filename, composed_name,
                                resourceKind, single=single)
                    if single:
                        # Break the loop to ensure the file is closed.
                        break
            # Ensure the old file is deleted
            if len(datas) > 1 and rename:
                os.remove(filename)


def ignored(filename, ignore_list: list):
    return any([ignore in filename for ignore in ignore_list])


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description='Rename some files.')
    parser.add_argument('--raise', dest='exception', action='store_true')
    parser.add_argument('--rename', dest='rename', action='store_true')
    parser.add_argument('--only-errors', dest='show_errors',
                        action='store_true')
    parser.add_argument('-i', '--ignore-list',
                        dest='ignore_list', nargs='+', help='List flag')
    parser.set_defaults(exception=False)
    parser.set_defaults(show_errors=False)
    parser.set_defaults(rename=False)
    parser.set_defaults(ignore_list=[])
    args = parser.parse_args()
    files = glob.glob('**', recursive=True)
    files = [filename for filename in files if Path(
        filename).is_file() and not ignored(filename, args.ignore_list)]
    validate(files, exception=args.exception,
             show_errors=args.show_errors, rename=args.rename)
