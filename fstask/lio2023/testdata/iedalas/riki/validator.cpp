#include <bits/stdc++.h>
#include "testlib.h"
using namespace std;

const int MAXT = 720;

int main(int argc, char* argv[]) {
    registerValidation(argc, argv);

	int T = inf.readInt(0, MAXT, "T"); inf.readEoln();
    inf.readEof();

    int hours = T / 12;
    int minutes = T % 60;

    if (validator.group() == "0")
    {
        inf.ensure(T == 131);
    }
    else if (validator.group() == "1")
    {
        inf.ensure(T % 10 == 0 && T % 60 != 0);
    }
    else if (validator.group() == "2")
    {
        inf.ensure(hours == 0 || minutes == 0);
    }
    else if (validator.group() == "3")
    {
        // No restrictions
    }
    else assert(false);
}
