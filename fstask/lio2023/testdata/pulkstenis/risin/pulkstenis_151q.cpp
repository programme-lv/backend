#include <iostream>
#include <vector>
#include <cstring>
using namespace std;


struct clock12
{
    int h, m, s;
    clock12 (int _h=0, int _m=0, int _s=0) {
        this->h = _h;
        this->m = _m;
        this->s = _s;
    }
    void inc() {
        this->s++;
        if (this->s == 60) {
            this->s = 0;
            this->m++;
        }
        if (this->m == 60) {
            this->m = 0;
            this->h++;
        }
        if (this->h == 12) {
            this->h = 0;
        }
    }
    int min_gap() {
        int th = (this->h)*5 + (this->m)/12, tm = this->m, ts = this->s;
        int ghm = abs(th-tm), gms = abs(tm-ts), gsh = abs(ts-th);
        int min_gap = min(ghm, 60-ghm);
        min_gap = min(min_gap, min(gms, 60-gms));
        min_gap = min(min_gap, min(gsh, 60-gsh));
        return min_gap;
    }
};


#define T_MAX (60*60*12)

int t_to_min_gap[T_MAX], gap0;
vector<int> ts_potential;

void calc_all_min_gaps()
{
    clock12 c = clock12(0, 0, 0);
    for (int t = 0; t < T_MAX; t++) {
        t_to_min_gap[t] = c.min_gap();
        c.inc();
    }
}

int get_next()
{
    int n_sample = 1000, cnt[60], cnt_best = 9999, P_best;
    for (int k = 0; k < n_sample; k++)
    {
        int P = rand() % T_MAX;
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

    int cnt = 1;
    while (true)
    {
        if (ts_potential.size() == 1) {
            while (cnt < 151)
            {
                cout << 0 << " " << cnt << endl;
                cnt++;
                int tmp;
                cin >> tmp;
            }
            cout << 1 << " " << ts_potential[0] << endl;
            break;
        }

        int gapq, Pq = get_next();
        cnt++;
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
