#!/bin/bash

python3 -m venv env || exit 0
source env/bin/activate
pip install -r requirements.txt
