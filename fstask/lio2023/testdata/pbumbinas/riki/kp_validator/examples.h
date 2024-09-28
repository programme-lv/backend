#include "testlib.h"
#include <vector>
#include "utils.h"

namespace examples
{
    void validate()
    {
        int N = inf.readInt();
        inf.readEoln();
        ensure(N >= 1 && N <= 3);
        std::vector<std::pair<int, long long>> queries;
        if (N == 1)
        {
            queries = {{0, 5}};
            utils::validate_queries(queries);
        }
        else if (N == 2)
        {
            queries = {{0, 9}, {3, 3}};
            utils::validate_queries(queries);
        }
        else
        { // N==3
            queries = {{0, 8}, {0, 7}, {0, 5}};
            utils::validate_queries(queries);
        }
    }
}