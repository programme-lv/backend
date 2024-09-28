#include "testlib.h"

#include <bits/stdc++.h>

using namespace std;

const int MAXT = 720;
 
int readAns(int T, InStream& in) {
    int cnt = 0;
    while (true) {
        int type = in.readInt(0, 1, "type");
        if (type == 0) {
            in.readInt(); // Participant query
            cnt++;
            in.readInt(); // Interactor response
        } else if (type == 1) {
            int Tguess = in.readInt(0, MAXT - 1);
            if (Tguess != T) {
                inf.quitf(_wa, "Incorrect guess");
            }
            break;
        } else {
            in.quitf(_fail, "Incorrect checker, validator");
        }
    }
    return cnt;
}

const int BEST_CNT = 5;
const int MAX_LIMIT = 100;

void score_solution(int pans) {
    if (pans > MAX_LIMIT)
    {
        quitf(_pc(0 * 2), "Correct, too many guesses!");
    }
    else if (pans <= 5)
    {
        quitf(_ok, "answer %d\n", pans);
    }
    else
    {
        float points = (2 / sqrt((double) (pans - 1))) * 100;
        auto integer_points = llround(points);
        if (integer_points <= 0 || integer_points >= 100)
        {
            quitf(_fail, "Interactor point calculation assertion error");
        }
        quitf(_pc(integer_points * 2), "Correct, too many guesses!");
    }
}
 
int main(int argc, char* argv[]) {
    registerTestlibCmd(argc, argv);
	
	int T = inf.readInt(0, MAXT - 1, "T");
 
    int jans = readAns(T, ans);
    int pans = readAns(T, ouf);

    if (jans > BEST_CNT)
    {
        quitf(_fail, "Incorrect jury solution!");
    }

    score_solution(pans);
}
