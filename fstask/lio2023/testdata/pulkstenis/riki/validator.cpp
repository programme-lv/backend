#include <bits/stdc++.h>
#include "testlib.h"
using namespace std;

const int MAXT = 43200;

int main(int argc, char* argv[]) {
    registerValidation(argc, argv);

	int T = inf.readInt(0, MAXT, "T"); inf.readEoln();

    int s = T % 60;
    int m = (T / 60) % 60;
    int h = (T / (60 * 12));
    ensure(s + m * 60 + (h / 5) * 60 * 60 == T);
    inf.readEof();

    set<int> a;
    a.insert({s, m, h});

    if (validator.group() == "0")
    {
        inf.ensure(T == 14379);
    }
    else if (validator.group() == "1")
    {
        inf.ensure(a.count(0) && a.count(30));
    }
    else if (validator.group() == "2")
    {
        inf.ensure(a.count(0));
    }
    else if (validator.group() == "3")
    {

    }
    else assert(false);
}
