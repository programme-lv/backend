#!/usr/bin/env python3

import generator
import random as r
from pathlib import Path
from enum import Enum

r.seed(42)

g = generator.TestGen("pbumbinas", "generator.cpp", "../risin/piramidas_pp_ok.cpp", "./testi")

class Type(Enum):
    AROUND_PEAK = 0
    RANDOM = 1

def bumbinas(type: Type, Q: int, B: int, 
             peak_count: int, peak_dist: int, 
             initial_hill_count: int, initial_hill_remainder: int,
             limitb: int):
    # Peak_dist > 0 -- randomize max
    # Peak_dist < 0 -- equalent distance
    # Limit b -- limit request size after initial_hill_count_queries
    randval = r.randrange(1000000)
    g.GenerateTest([
                       type.value, Q, B,
                       peak_count, peak_dist,
                       initial_hill_count, initial_hill_remainder,
                       limitb,
                       randval
                   ])

g.NewGroup(0, "0. apakšuzdevums", public=True)
g.CopyRawTest("./special_tests/s1.txt")
g.CopyRawTest("./special_tests/s2.txt")
g.CopyRawTest("./special_tests/s3.txt")

# 2 punkti

g.NewGroup(2, "1. apakšuzdevums", public=True)
g.CopyRawTest("./special_tests/ap1.txt")
bumbinas(Type.RANDOM, 7, 100, 4, 0, 4, 0, 0)
bumbinas(Type.AROUND_PEAK, 8, 100, 4, 0, 0, 0, 0)

# 8punkti  B <= 10^4

g.NewGroup(2, "2. apakšuzdevums", public=True)

g.CopyRawTest("./special_tests/small1.txt")
g.CopyRawTest("./special_tests/small2.txt")

Q = 10**4
B = 10**4

bumbinas(Type.RANDOM, Q, B, 0, 0, 0, 0, 0)
bumbinas(Type.RANDOM, Q // 10, B, 0, 0, 0, 0, 0)

for B in [B // 100, B]:
    g.NewGroup(3)
    bumbinas(Type.RANDOM, min(Q, B), B, 0, 0, 0, 0, 0)
    bumbinas(Type.AROUND_PEAK, min(Q, B), B, 1, 40, 0, 0, 0)
    bumbinas(Type.AROUND_PEAK, min(Q, B), B, 2, 100, 0, 0, 0)
    bumbinas(Type.AROUND_PEAK, min(Q, B), B, 2, -20, 0, 0, 0)

# 10punkti  B <= 10^8 , Q <= 1000

g.NewGroup(2, "3. apakšuzdevums", public=True)

Q = 10**3
B = 10**8

bumbinas(Type.RANDOM, Q, B, 0, 0, 0, 0, 0)
bumbinas(Type.RANDOM, Q, B, 0, 0, 0, 0, 0)

for B in [B // 10, B]:
    g.NewGroup(2)
    bumbinas(Type.RANDOM, Q, B, 0, 0, 0, 0, 0)
    bumbinas(Type.AROUND_PEAK, Q, B, 1, 0, 0, 0, 0)
    bumbinas(Type.AROUND_PEAK, Q, B, 2, 30000, 8, 0, 3)
    g.NewGroup(2)
    bumbinas(Type.AROUND_PEAK, Q, B, 3, -20, 0, 0, 0)
    bumbinas(Type.AROUND_PEAK, Q, B, 2, -20, 4, 0, 0)
    bumbinas(Type.AROUND_PEAK, Q, B, 3, -20, 10, 0, 0)
    bumbinas(Type.AROUND_PEAK, Q, B, 2, 30000, 8, 0, 3)


# 10punkti  B <= 10^10 , Q <= 100

g.NewGroup(2, "4. apakšuzdevums", public=True)

Q = 10**2
B = 10**10

bumbinas(Type.RANDOM, Q, B, 0, 0, 0, 0, 0)
bumbinas(Type.RANDOM, Q, B, 0, 0, 0, 0, 0)

for B in [B // 10, B]:
    g.NewGroup(2)
    bumbinas(Type.RANDOM, Q, B, 0, 0, 0, 0, 0)
    bumbinas(Type.AROUND_PEAK, Q, B, 1, 0, 0, 0, 0)
    bumbinas(Type.AROUND_PEAK, Q, B, 2, 300, 0, 0, 0)
    bumbinas(Type.AROUND_PEAK, Q, B, 2, 30000, 4, 0, 3)
    g.NewGroup(2)
    bumbinas(Type.AROUND_PEAK, Q, B, 3, 300, 0, 0, 0)
    bumbinas(Type.AROUND_PEAK, Q, B, 2, 2000, 4, 0, 0)
    bumbinas(Type.AROUND_PEAK, Q, B, 4, 20000, 10, 0, 0)
    bumbinas(Type.AROUND_PEAK, Q, B, 2, 30000, 4, 0, 3)


# 15punkti  B <= 10^12, no 11 metiena bi <= 1
g.NewGroup(3, "5. apakšuzdevums", public=True)

Q = 10**4
B = 10**12
LIMITB = 1

bumbinas(Type.RANDOM, Q, B, 0, 0, 10, 0, LIMITB)
bumbinas(Type.RANDOM, Q, B, 0, 0, 10, 0, LIMITB)

for B in [B // 10, B // 2, B]:
    g.NewGroup(2)
    bumbinas(Type.RANDOM, Q, B, 0, 0, 10, 0, LIMITB)
    bumbinas(Type.AROUND_PEAK, Q, B, 1, 0, 10, 0, LIMITB)
    bumbinas(Type.AROUND_PEAK, Q, B, 1, 0, 10, 0, LIMITB)
    bumbinas(Type.AROUND_PEAK, Q, B, 1, -100000, 10, 0, LIMITB)
    g.NewGroup(2)
    bumbinas(Type.AROUND_PEAK, Q, B, 3, 30000, 10, 0, LIMITB)
    bumbinas(Type.AROUND_PEAK, Q, B, 2, 2000, 4, 0, LIMITB)
    bumbinas(Type.AROUND_PEAK, Q, B, 4, 20000, 10, 0, LIMITB)


# 15punkti  B <= 10^12, no 11 metiena bi <= 1000

g.NewGroup(3, "6. apakšuzdevums", public=True)
Q = 10**4
B = 10**12
LIMITB = 1000

bumbinas(Type.RANDOM, Q, B, 0, 0, 10, 0, LIMITB)
bumbinas(Type.RANDOM, Q, B, 0, 0, 10, 0, LIMITB)

for B in [B // 10, B // 2, B]:
    g.NewGroup(2)
    bumbinas(Type.RANDOM, Q, B, 10, 0, 10, 0, LIMITB)
    bumbinas(Type.AROUND_PEAK, Q, B, 1, 0, 10, 0, LIMITB)
    bumbinas(Type.AROUND_PEAK, Q, B, 2, 30000, 10, 0, LIMITB)
    g.NewGroup(2)
    bumbinas(Type.AROUND_PEAK, Q, B, 3, 500, 10, 0, LIMITB)
    bumbinas(Type.AROUND_PEAK, Q, B, 2, -20, 4, 0, LIMITB)
    bumbinas(Type.AROUND_PEAK, Q, B, 3, -20, 10, 0, LIMITB)


# 25punkti  B <= 10^12,

g.NewGroup(5, "7. apakšuzdevums", public=True)
Q = 10**4
B = 10**12

bumbinas(Type.RANDOM, Q, B, 0, 0, 10, 0, 0)
bumbinas(Type.RANDOM, Q, B, 0, 0, 10, 0, 0)

for B in [B // 100, B // 10, B // 2, B]:
    g.NewGroup(3)
    bumbinas(Type.RANDOM, Q, B, 0, 0, 0, 0, 0)
    bumbinas(Type.RANDOM, Q, B, 0, 0, 0, 0, 0)
    bumbinas(Type.AROUND_PEAK, Q, B, 1, 0, 0, 0, 0)
    bumbinas(Type.AROUND_PEAK, Q, B, 1, 0, 10, 0, 1000)
    bumbinas(Type.AROUND_PEAK, Q, B, 10, 100000, 100, 0, 1000000000)
    g.NewGroup(2)
    bumbinas(Type.AROUND_PEAK, Q, B, 3, 3000, 10, 0, 0)
    bumbinas(Type.AROUND_PEAK, Q, B, 7, -2000, 10, 0, 0)
    bumbinas(Type.AROUND_PEAK, Q, B, 10, 100000, 100, 0, 1000000000)


# 15punkti  B <= 10^18
g.NewGroup(3, "8. apakšuzdevums", public=True)
Q = 10**4
B = 10**18

bumbinas(Type.RANDOM, Q, B // 10, 0, 0, 10, 0, 0)
bumbinas(Type.RANDOM, Q, B // 10, 0, 0, 10, 0, 0)
bumbinas(Type.RANDOM, Q, B // 10, 0, 0, 0, 0, 0)

for B in [B // 100, B // 10, B]:
    g.NewGroup(2)
    bumbinas(Type.RANDOM, Q, B, 0, 0, 0, 0, 0)
    bumbinas(Type.RANDOM, Q, B, 0, 0, 0, 0, 0)
    bumbinas(Type.AROUND_PEAK, Q, B, 1, 0, 0, 0, 0)
    bumbinas(Type.AROUND_PEAK, Q, B, 10, 100000, 100, 0, 1000000000)
    g.NewGroup(2)
    bumbinas(Type.AROUND_PEAK, Q, B, 2, 30, 4, 0, 0)
    bumbinas(Type.AROUND_PEAK, Q, B, 3, 30000, 10, 0, 0)
    bumbinas(Type.AROUND_PEAK, Q, B, 7, -15000, 10, 0, 0)
    bumbinas(Type.AROUND_PEAK, Q, B, 10, 100000, 100, 0, 1000000000)

g.CopyRawTest("./special_tests/huge1.txt")
g.CopyRawTest("./special_tests/huge2.txt")
g.CopyRawTest("./special_tests/huge3.txt")


g.GeneratePointFile(Path("./punkti.txt"))
g.GenerateTestDescription(Path("./description.txt"))
g.GenerateTestZip(Path("./testi.zip"))
g.GenerateTestZip(Path("./itesti.zip"), include_output=False)
g.End()
