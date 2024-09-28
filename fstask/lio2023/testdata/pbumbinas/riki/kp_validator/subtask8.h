#include "testlib.h"

namespace subtask8
{
    void validate()
    {
        int N = inf.readInt(1,1e4);
        inf.readEoln();
        long long B_sum = 0;
        for(int i=0;i<N;i++) {
            inf.readInt(-1e6,1e6);
            inf.readSpace();
            long long bi = inf.readLong(1ll,1'000'000'000'000'000'000);
            inf.readEoln();
            B_sum += bi;
            ensure(B_sum<=1e18);
        }
    }
}