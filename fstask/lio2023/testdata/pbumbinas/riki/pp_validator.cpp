#include <bits/stdc++.h>
#include "testlib.h"
using namespace std;

using ll = long long;
using ull = unsigned long long;
const ll MAXB = 1'000'000'000'000'000'000LL;
const int MAXC = 1'000'000;
const int MAXN = 10'000;

int main(int argc, char* argv[]) {
    registerValidation(argc, argv);

	ll n = inf.readInt(1, MAXN, "n"); inf.readEoln();
    ll total = 0;
    ll max_from_10 = 0;
    for (int i = 1; i <= n; i++)
    {
        int pos = inf.readInt(-MAXC, MAXC, "p"); inf.readSpace();
        ll val = inf.readLong(1, MAXB, "b"); inf.readEoln();
        total += val;
        if (i > 10)
        {
            max_from_10 = max(max_from_10, val);
        }
    }
    inf.readEof();

    inf.ensuref(total <= MAXB, "Total b");

    if (validator.group() == "0") {

    }
    else if (validator.group() == "1") 
    {

    } 
    else if (validator.group() == "2") 
    {
        inf.ensure(total <= 10000);
    } 
    else if (validator.group() == "3") 
    {
        inf.ensure(total <= 10000'0000 && n <= 1000);
    } 
    else if (validator.group() == "4") 
    {
        inf.ensure(total <= 10000'0000'00 && n <= 100);
    } 
    else if (validator.group() == "5") 
    {
        inf.ensure(total <= 10000'0000'0000 && max_from_10 == 1);
    } 
    else if (validator.group() == "6") 
    {
        inf.ensure(total <= 10000'0000'0000 && max_from_10 <= 1000);
    } 
    else if (validator.group() == "7") 
    {
        inf.ensure(total <= 10000'0000'0000);
    } 
    else if (validator.group() == "8") 
    {

    } 
    else 
    {
        assert(false);
    }
}
