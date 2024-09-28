#include "testlib.h"

#include <bits/stdc++.h>

using namespace std;

using ll = long long;

const int MAXN = 200000;
const int MAXV = 1'000'000'000;

int main(int argc, char* argv[]) {
    registerValidation(argc, argv);

    int N = inf.readInt(2, MAXN, "N"); inf.readEoln();
    auto a = inf.readInts(N, 1, MAXV, "ai"); inf.readEoln();
    inf.readEof();

    set<int> seta(a.begin(), a.end());

    bool size103 = all_of(a.begin(), a.end(), 
                          [](int val) { return val <= 1000; }
                         );
    bool size106 = all_of(a.begin(), a.end(), 
                          [](int val) { return val <= 1000000; }
                         );

    if (validator.group() == "0")
    {
        inf.ensure(
            a == vector<int>({12, 11, 9, 7, 7, 4, 1, 2, 4}) ||
            a == vector<int>({1, 1, 200, 1}) ||
            a == vector<int>({2, 33, 33, 1}) 
        );
    } 
    else if (validator.group() == "1")
    {
        inf.ensure(
            a == vector<int>({1, 26, 13, 10, 7, 5, 2, 14}) ||
            a == vector<int>({11, 3, 2, 2, 6, 1, 4, 5, 9, 12, 6, 1, 4, 5, 9, 12, 7, 8, 10}) ||
            a == vector<int>({1, 26, 1, 2, 3, 4, 5, 13, 10, 15, 15, 15, 7, 5, 2, 14, 12, 11, 10, 15, 23, 20, 19, 5}) 
        );
    } 
    else if (validator.group() == "2")
    {
        inf.ensure(N <= 16 && size103);
    }
    else if (validator.group() == "3")
    {
        inf.ensure(16 < N && N <= 1024 && size106 && seta.size() == a.size());
    }
    else if (validator.group() == "4")
    {
        // No restrictions
    }
    else 
    {
        assert(false);
    }
}
