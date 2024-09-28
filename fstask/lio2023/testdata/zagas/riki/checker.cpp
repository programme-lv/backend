#include "testlib.h"

#include <bits/stdc++.h>

using namespace std;

const int MAXV = 1'000'000'000;
int N;
vector<int> a;
 
int readAns(InStream& in) {
    map<int, int> cntelem;
    for (auto t : a)
    {
        cntelem[t] += 1;
    }

    int S = in.readInt(0, MAXV, "S");

    int prev = 2 * MAXV;
    int mindist = 2 * MAXV;

    for (int i = 0; i < N; i++)
    {
        int num = in.readInt();

        in.ensuref(cntelem[num] > 0, "Number %d not available", num);
        cntelem[num]--;

        if (S > 0)
        {
            if ((i % 2) == 0)
            {
                in.ensuref(prev > num, "Incorrect sequence %d %d", prev, num);
            }
            else
            {
                in.ensuref(prev < num, "Incorrect sequence %d %d", prev, num);
            }

            mindist = min(abs(prev - num), mindist);
            prev = num;
        }
    }

    if (S > 0)
    {
        in.ensuref(mindist == S, "S doesn't match calculated distance from proof (%d != %d)", S, mindist);
    }

    return S;
}
 
int main(int argc, char* argv[]) {
    registerTestlibCmd(argc, argv);
	
    N = inf.readInt();
    for (int i = 0; i < N; i++)
    {
        a.push_back(inf.readInt(1, MAXV, "a_i"));
    }
 
    auto jans = readAns(ans);
    auto pans = readAns(ouf);

    if (jans > pans)
    {
        quitf(_wa, "Jury has better solution");
    }
    else if (jans == pans)
    {
        quitf(_ok, "Correct");
    }
    else
    {
        quitf(_fail, "Incorrect jury solution!");
    }
}
