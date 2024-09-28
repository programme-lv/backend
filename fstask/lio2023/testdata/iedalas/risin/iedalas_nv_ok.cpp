#include <iostream>
#include <vector>
#include <cstring>
using namespace std;


struct clock12
{
    int h, m;
    clock12 (int _h=0, int _m=0) {
        this->h = _h;
        this->m = _m;
    }
    void inc() {
        this->m++;
        if (this->m == 60) {
            this->m = 0;
            this->h++;
        }
        if (this->h == 12) {
            this->h = 0;
        }
    }
    int min_gap() {
        int th = (this->h)*5 + (this->m)/12, tm = this->m;
        int g = abs(th-tm);
        int min_gap = min(g, 60-g);
        return min_gap;
    }
};


#define T_MAX (60*12)

int t_to_min_gap[T_MAX], gap0;
vector<int> ts_potential;

void calc_all_min_gaps()
{
    clock12 c = clock12(0, 0);
    for (int t = 0; t < T_MAX; t++) {
        t_to_min_gap[t] = c.min_gap();
        c.inc();
    }
}

int get_next()
{
    int cnt[60], cnt_best = 9999, P_best;
    for (int P = 0; P < T_MAX; P++)
    {
        memset(cnt, 0, sizeof(cnt));
        for (int i = 0; i < ts_potential.size(); i++) {
            int t = (ts_potential[i] + P) % T_MAX;
            cnt[ t_to_min_gap[t] ]++;
        }
        int cnt_max = 0;
        for (int g = 0; g < 60; g++) {
            cnt_max = max(cnt_max, cnt[g]);
        }
        if (cnt_best > cnt_max) {
            cnt_best = cnt_max;
            P_best = P;
        }
    }
    return P_best;
}

void solve()
{
    ts_potential.clear();
    for (int t = 0; t < T_MAX; t++) {
        if (t_to_min_gap[t] == gap0) {
            ts_potential.push_back(t);
        }
    }

    while (true)
    {
        if (ts_potential.size() == 1) {
            cout << 1 << " " << ts_potential[0] << endl;
            break;
        }

        int gapq, Pq = get_next();
        cout << 0 << " " << Pq << endl;
        cin >> gapq;
        vector<int> ts_potential_tmp;
        for (int i = 0; i < ts_potential.size(); i++) {
            int t = (ts_potential[i] + Pq) % T_MAX;
            if (t_to_min_gap[t] == gapq) ts_potential_tmp.push_back(ts_potential[i]);
        }
        ts_potential = ts_potential_tmp;
    }
}


int main()
{
    calc_all_min_gaps();
    cout << "0 0" << endl;
    cin >> gap0;
    solve();
//    while (cin >> gap0) {
//        solve();
//    }
}
