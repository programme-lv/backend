#include <bits/stdc++.h>
#include "testlib.h"
using namespace std;

int main(int argc, char* argv[]) {
    registerValidation(argc, argv);

	int n = inf.readInt(1, 500, "n"); inf.readEoln();
    inf.readEof();

	if (validator.group() == "0") {
        inf.ensure(n == 6);
    } else if (validator.group() == "1") {
        inf.ensure(1 <= n && n <= 5);
    } else if (validator.group() == "2") {
        inf.ensure(6 <= n && n <= 80);
    } else if (validator.group() == "3") {
        inf.ensure(81 <= n && n <= 400);
    } else if (validator.group() == "4") {
        inf.ensure(401 <= n && n <= 500);
    }
}
