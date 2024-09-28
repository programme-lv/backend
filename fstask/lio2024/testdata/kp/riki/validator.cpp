#include <bits/stdc++.h>
#include <sys/resource.h>
#include "testlib.h"
using namespace std;

//const int MAXN = 1'000, MAXM = 1'000;
const int MAXN = 1'000'000, MAXM = 1'000'000;

int main(int argc, char* argv[]) {
    registerValidation(argc, argv);

	int n = inf.readInt(2,MAXN,"n"); inf.readSpace();
	int m = inf.readInt(2,MAXM,"m"); inf.readSpace();
	int k = inf.readInt(1,min(n, m),"n");
	inf.readEoln();
	for (int i = 1; i <= n; i++) {
        for (int j = 1; j <= m; j++) {
            char c = inf.readChar();
            inf.ensure(c == '.' || c == 'X' || c == 'A' || c == 'B');
            if (c == 'A') {
                inf.ensuref(((i+k-1 <= n) && (j+k-1 <= m)), "KP initial position A does not fit in frame");
            }
        }
        inf.readEoln();
    }
    inf.readEof();

	if (validator.group() == "0") {
        inf.ensure(n == 5 && m == 9 && k == 3);
    } else if (validator.group() == "1") {
        inf.ensure((n == 5 && m == 5 && k == 3) ||
                   (n == 6 && m == 3 && k == 1) ||
                   (n == 6 && m == 4 && k == 2));
    } else if (validator.group() == "2") {
        inf.ensure(n*m <= 1'000);
    } else if (validator.group() == "3") {
        inf.ensure(n <= 1000);
        inf.ensure(m <= 1000);
    } else if (validator.group() == "4") {
        inf.ensure(1'000 < n*m && n*m <= 1'000'000);
    }
}
