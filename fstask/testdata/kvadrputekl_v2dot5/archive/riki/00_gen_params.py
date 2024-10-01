#! /usr/bin/python

import pathlib
import csv
from random import randint
from math import sqrt

subtasks = [
    {"NM_min": 4, "NM_max": 50, "tests_per_group": 3, "groups": 1, "subtask": 1},
    
    {"NM_min": 10, "NM_max": 100, "tests_per_group": 3, "groups": 2, "subtask": 2},

    {"NM_min": 10, "NM_max": 1000, "tests_per_group": 3, "groups": 3, "subtask": 2},
    
    {"NM_min": 10, "NM_max": 100000, "tests_per_group": 3, "groups": 2, "subtask": 3},

    {"NM_min": 1000, "NM_max": 1000000, "tests_per_group": 3, "groups": 3, "subtask": 3},

    {"NM_min": 1000000, "NM_max": 1000000, "tests_per_group": 4, "groups": 1, "subtask": 3}
]

params = []
params.append(("N", "M", "K", "OK", "T", "G", "ST"))

group = 0
for i in range(0, len(subtasks)):
    groups = subtasks[i]["groups"]
    for j in range(groups):
        group += 1  # 5 groups of 3 tests per each subtask
        tests_per_group = subtasks[i]["tests_per_group"]
        for k in range(tests_per_group):
            subtask = subtasks[i]
            NM_1 = randint(subtask["NM_min"], subtask["NM_max"])
            NM_2 = randint(subtask["NM_min"], subtask["NM_max"])
            NM = max(NM_1, NM_2)
            def gen_params(area) -> (int, int, int, int, int):
                N = randint(1, int(sqrt(area)))
                if randint(0, 1) == 0:
                    N = int(sqrt(area))
                M = area//N
                if randint(0, 1) == 0:
                    N, M = M, N
                K = randint(1, min(N, M))
                OK = randint(0, 3)
                if OK > 0:
                    OK = 1
                return N, M, K, OK
            N, M, K, OK = gen_params(NM)
            while OK == 0 and K == min(N, M) or N<2 or M<2:
                N, M, K, OK = gen_params(NM)
            T = "acdb"
            if randint(0,2)==0 and OK:
                T = "sparse"
            params.append((N, M, K, OK, T, f"{str(group).zfill(2)}{chr(k+ord('a'))}", subtask["subtask"]))

dir = pathlib.Path(__file__).parent.absolute()
csv_path = dir.joinpath("01_param_list.csv")
with open(csv_path, "w") as f:
    writer = csv.writer(f, delimiter="\t")
    writer.writerows(params)
