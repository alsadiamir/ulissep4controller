#!/bin/bash

start=$(date)
echo "start $start" >> times.txt
dataset_type=IDS2017

model=/home/prince7/ulissep4controller/lucid-cnn/10t-10n-IDS2017-LUCID-p4-training-dimezzato.h5

python3 lucid_cnn.py --predict_live localhost:9090 --model $model  --dataset_type $dataset_type

echo "end: $(date)"
echo "end: $(date)" >>  times.txt

