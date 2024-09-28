#include "testlib.h"
#include "utils.h"

namespace subtask1
{
    void validate()
    {
        int N = inf.readInt();
        inf.readEoln();
        ensure(N == 3 || N == 7 || N == 8);
        std::vector<std::pair<int, long long>> queries;
        if (N == 3)
        {
            queries = {
                {-1000000, 3},
                {1000000, 5},
                {0, 5}};
            utils::validate_queries(queries);
        }
        else if (N == 7)
        {
            queries = {
                {1607, 13},
                {2415, 55},
                {1607, 11},
                {2415, 17},
                {760194, 1},
                {-12164, 1},
                {330520, 2},
            };
            utils::validate_queries(queries);
        }
        else
        { // N==3
            queries = {
                {2019, 8},
                {2019, 11},
                {788, 1},
                {788, 4},
                {1689, 23},
                {2028, 4},
                {2019, 6},
                {765, 43}};
            utils::validate_queries(queries);
        }
    }
}