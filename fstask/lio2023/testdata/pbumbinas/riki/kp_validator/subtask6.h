#include "testlib.h"

namespace subtask6
{
    void validate()
    {
        int N = inf.readInt(1,1e4);
        inf.readEoln();
        long long B_sum = 0;
        for(int i=0;i<N;i++) {
            inf.readInt(-1e6,1e6);
            inf.readSpace();
            long long bi;
            if(i<10)
                bi = inf.readLong(1ll,1'000'000'000'000'000'000);
            else
                bi = inf.readLong(1ll,1000ll);
            inf.readEoln();
            B_sum += bi;
            ensure(B_sum<=1e12);
        }
    }
}