#pragma once
#include <vector>
#include "testlib.h"

namespace utils{
    bool validate_queries(std::vector<std::pair<int,long long>> queries) {
        for(int i=0;i<queries.size();i++) {
            inf.readInt(queries[i].first,queries[i].first);
            inf.readSpace();
            inf.readLong(queries[i].second,queries[i].second);
            inf.readEoln();
        }
        return true;
    }
}