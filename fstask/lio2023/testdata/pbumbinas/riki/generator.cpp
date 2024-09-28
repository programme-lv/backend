#include "testlib.h"
#include <bits/stdc++.h>

using namespace std;

using ll = long long;

vector<pair<int, ll>> output;

const int MAXQ = 10000;
const int MAXC = 1000000;
const ll MAXB = 1000'000'000'000'000'000LL;

int TOTALQ;
ll TOTALB;
ll LIMITB;
ll INITIAL_HILL_COUNT;

void output_test() {
    assert(MAXQ >= (int)output.size() && TOTALQ >= (int)output.size());
    printf("%d\n", (int)output.size());
    ll total = 0;
    int idx = 0;
    for (auto t : output)
    {
        idx++;
        assert(!(LIMITB && idx > INITIAL_HILL_COUNT && t.second > LIMITB));
        assert(abs(t.first) <= MAXC);
        printf("%d %lld\n", t.first, t.second);
        assert(t.second > 0);
        assert(t.second <= TOTALB);
        total += t.second;
    }
    assert(total <= TOTALB && total <= MAXB);
}

int main(int argc, char *argv[])
{
    registerGen(argc, argv, 1);

    assert(argc >= 8);
    int type = stoi(argv[1]);
    TOTALQ = stoi(argv[2]);
    TOTALB = stoll(argv[3]);
    int peak_cnt = stoi(argv[4]);
    int peak_dist = stoi(argv[5]);
    INITIAL_HILL_COUNT = stoi(argv[6]);
    int initial_hill_remainder = stoi(argv[7]);
    LIMITB = stoll(argv[8]);
    assert(TOTALB >= TOTALQ);

    ll REMAININGB = TOTALB;
    ll REMAININGQ = TOTALQ;

    // Random one pyramid
    int center_pos = rnd.next(-1000, 1000);
    vector<int> peaks;
    if (peak_dist >= 0)
    {
        if (peak_dist == 0)
                peak_dist = 1000;
        peaks.push_back(center_pos);
        for (int i = 2; i <= peak_cnt; i++)
        {
            int next_pos = peaks.back() + rnd.wnext(peak_dist, 2);
            peaks.push_back(next_pos);
        }
        if (rnd.next(0, 1) == 1)
        {
            reverse(peaks.begin(), peaks.end());
        }
    }
    else
    {
        peak_dist = -peak_dist;
        int total_dist = peak_dist * (peak_cnt - 1);
        center_pos = -total_dist / 2;
        peaks.push_back(center_pos);
        for (int i = 2; i <= peak_cnt; i++)
        {
            peaks.push_back(peaks.back() + peak_dist);
        }
        if (rnd.next(0, 1) == 1)
        {
            reverse(peaks.begin(), peaks.end());
        }
    }
    
    if (INITIAL_HILL_COUNT > 0)
    {
        ll value;
        if (initial_hill_remainder)
        {
            value = REMAININGB - initial_hill_remainder;
        }
        else if (LIMITB)
        {
            ll maxval = REMAININGB - 1 * (REMAININGQ - INITIAL_HILL_COUNT);
            ll minval = max(REMAININGB - (REMAININGQ - INITIAL_HILL_COUNT) * LIMITB, (ll)INITIAL_HILL_COUNT);
            assert(maxval >= minval);
            value = rnd.wnext(minval, maxval, -10);
        }
        else
        {
            ll available_value = REMAININGB - (REMAININGQ - INITIAL_HILL_COUNT);
            value = rnd.wnext((ll)INITIAL_HILL_COUNT, available_value, 10);
        }
        assert(value >= INITIAL_HILL_COUNT);

        for (auto t: rnd.partition(INITIAL_HILL_COUNT, value))
        {
            REMAININGQ--;
            REMAININGB -= t;
            output.push_back({rnd.any(peaks) + rnd.next(-1, 1), t});
        }
        assert(REMAININGB >= REMAININGQ);
    }

    auto partition = rnd.partition(REMAININGQ, REMAININGB);
    if (LIMITB > 0)
    {
        assert(REMAININGB >= 1 * REMAININGQ && REMAININGB <= REMAININGQ * LIMITB);
        partition.clear();

        while (REMAININGQ > 0)
        {
            ll maxval = min((ll)LIMITB, REMAININGB - (REMAININGQ - 1));
            ll minval = max(REMAININGB - (REMAININGQ - 1) * LIMITB, 1LL);
            assert(minval <= maxval);

            ll val = rnd.next(minval, maxval);
            assert(1 <= val && val <= LIMITB);
            partition.push_back(val);
            REMAININGB-=val;
            REMAININGQ--;
        }
    }

    for (auto t : partition)
    {
        if (type == 0)
        {
            int pos = rnd.any(peaks);
            pos = pos + rnd.wnext(0, 50, -3) * rnd.next(-1, 1);
            output.push_back({pos, t});
        }
        else if (type == 1)
        {
            int pos = rnd.next(-MAXC + 10, MAXC - 10);
            output.push_back({pos, t});
        }
        else
        {
            assert(false);
        }
    }

    output_test();
}
