#!/usr/bin/env python3

import generator
import random as r
from pathlib import Path
from math import isqrt
from typing import Tuple, List

r.seed(42)

def distance(a: int, b: int):
    dist1 = abs(a-b)
    dist2 = 60 - dist1
    return min(dist1, dist2)

def position(t: int) -> Tuple[int, int]:
    assert(0 <= t < 720)
    minutes = t % 60
    hours = t // 12
    return (minutes, hours)

def read_input():
    with open("./interval.txt") as file:
        lines = [int(line.rstrip()) for line in file]
    t1 = []
    t2 = []
    t3 = []
    for t in lines:
        assert(0 <= t and t < 720)
        minutes_pos, hours_pos = position(t)
        if t % 10 == 0 and t % 60 != 0:
            t1.append(t)
        elif minutes_pos == 0 or hours_pos == 0:
            t2.append(t)
        else:
            t3.append(t)
    r.shuffle(t1)
    r.shuffle(t2)
    r.shuffle(t3)
    return t1, t2, t3

t1, t2, t3 = read_input()
print(len(t1), len(t2), len(t3))

g = generator.TestGen("iedalas", "generator.cpp", "./fake_output.cpp", "./testi")

def testgen(arr: List[int]):
    assert(len(arr) > 0)
    T = arr.pop(0)
    g.GenerateTest([
                       T
                   ])

g.NewGroup(0, "0. apakšuzdevums", public=True)
testgen([131])

# 20 punkti
g.NewGroup(4, "1. apakšuzdevums", public=True)
testgen(t1)
testgen(t1)
testgen(t1)
testgen(t1)


for _ in range(4):
    g.NewGroup(4)
    for _ in range(4):
        testgen(t1)

assert(len(t1)==1)
testgen(t1)
assert(len(t1)==0)

# 20punkti

g.NewGroup(4, "2. apakšuzdevums", public=True)
testgen(t2)
testgen(t2)
testgen(t2)
testgen(t2)


for _ in range(4):
    g.NewGroup(4)
    for _ in range(4):
        testgen(t2)

assert(len(t2)==1)
testgen(t2)
assert(len(t2)==0)

# 60punkti

g.NewGroup(4, "3. apakšuzdevums", public=True)
testgen(t3)
testgen(t3)
testgen(t3)
testgen(t3)


for _ in range(14):
    g.NewGroup(4)
    for _ in range(4):
        testgen(t3)

assert(len(t3)==1)
testgen(t3)
assert(len(t3)==0)

g.GeneratePointFile(Path("./punkti.txt"))
g.GenerateTestDescription(Path("./description.txt"))
g.GenerateTestZip(Path("./testi.zip"))
g.GenerateTestZip(Path("./itesti.zip"), include_output=False)
g.End()
