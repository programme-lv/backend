#include "testlib.h"
#include <queue>
using namespace std;

/*
N<Eoln>
a1<Space>a2<Space>...<Space>an<Eoln>
<Eof>

a1, a2, ..., an are distinct integers from 1 to N

1. three given tests
2. increasing sequence
3. n <= 10
4. M = N (augstākais tornis sastāvēs no visām N ripām)
5. n <= 3000
6. bez papildus ierobežojumiem
 
*/
using ll = int;
using ii = pair<ll,ll>;

const ll MAXN = 5e5;
ll l[MAXN], r[MAXN];
ii t[MAXN], history[MAXN];
ll interval_idx[MAXN];
ll h_length[MAXN];
bool removed[MAXN];
bool interesting[MAXN];

pair<ll,vector<ii>> solve(vector<ll> a){
    ll N = a.size();
    for(ll i=0;i<N;i++) cin>>a[i], a[i]--;
    for(ll i=0;i<N;i++){
        l[i]=r[i]=-1;
        if(i) l[i]=i-1;
        if(i<N-1) r[i]=i+1;
    }
    ll h_idx=0;
    for(ll i=0;i<N;i++) t[i]={a[i],a[i]};
    queue<ll> q;
    for(ll i=0;i<N;i++){
        if(i-1>=0&&a[i-1]-a[i]==1)
            q.push(i);
        if(i+1<N&&a[i+1]-a[i]==1)
            q.push(i);
    }
    for(ll i=0;i<N;i++) removed[i]=0;
    while(!q.empty()){
        ll f=q.front(); q.pop();
        if(f==-1) continue;
        if(removed[f]) continue;
        if(l[f]!=-1){
            if(t[f].first+1==t[l[f]].second){
                history[h_idx++]={t[f].second,t[l[f]].second};
                t[l[f]].second=t[f].second;
                if(r[f]!=-1)
                    l[r[f]]=l[f];
                r[l[f]]=r[f];
                removed[f]=1;
                q.push(l[l[f]]);
                q.push(l[f]);
                q.push(r[f]);
            }
        }
        if(r[f]!=-1){
            if(t[f].first+1==t[r[f]].second){
                history[h_idx++]={t[f].second,t[r[f]].second};
                t[r[f]].second=t[f].second;
                if(l[f]!=-1)
                    r[l[f]]=r[f];
                l[r[f]]=l[f];
                removed[f]=1;
                q.push(l[f]);
                q.push(r[f]);
                q.push(r[r[f]]);
            }
        }
    }
    ll mx_height=-1;
    for(ll i=0;i<N;i++){
        if(removed[i]) continue;
        if(t[i].first-t[i].second+1>mx_height)
            mx_height=t[i].first-t[i].second+1;
    }

    for(ll i=0;i<N;i++){
        if(removed[i]) continue;
        for(ll j=t[i].second;j<=t[i].first;j++)
            interval_idx[j]=i;
    }
    for(ll i=0;i<N;i++) h_length[i]=0;
    for(ll i=0;i<h_idx;i++){
        h_length[interval_idx[history[i].first]]++;
        h_length[interval_idx[history[i].second]]++;
    }

    ll res_idx = -1;
    for(ll i=0;i<N;i++){
        if(removed[i]) continue;
        if(t[i].first-t[i].second+1==mx_height&&(res_idx==-1||h_length[i]<=h_length[res_idx]))
            res_idx=i;
    }

    for(ll i=0;i<N;i++) interesting[i]=0;
    for(ll i=t[res_idx].second;i<=t[res_idx].first;i++)
        interesting[i]=1;
    
    vector<ii> res_h;
    for(ll i=0;i<h_idx;i++){
        if(interesting[history[i].first]||interesting[history[i].second]){
            res_h.push_back(history[i]);
        }
    }

    return {mx_height,res_h};
}

int main(int argc, char* argv[]) {
    registerValidation(argc, argv);

    ll N = inf.readInt(1, 500'000, "N");
    inf.readEoln();
    vector<ll> a(N);
    for (int i = 0; i < N; i++) {
        int a_i=inf.readInt(1, N, "a");
        a[i]=a_i;
        if (i < N - 1) {
            inf.readSpace();
        }
    }
    inf.readEoln();
    inf.readEof();

	if (validator.group() == "1") {
        vector<ll> r[3] = {
            {2, 6, 4, 5, 1, 3, 7},
            {3, 2, 5, 4, 7, 6, 1, 8},
            {4, 6, 5, 7, 9, 8, 2, 1, 3}
        };
        inf.ensure(a==r[0]||a==r[1]||a==r[2]);
    } else if (validator.group() == "2") {
        for (int i = 1; i < N; i++) {
            inf.ensure(a[i-1] < a[i]);
        }
    } else if (validator.group() == "3") {
        inf.ensure(N <= 10);
    } else if (validator.group() == "4") {
        auto r = solve(a);
        inf.ensure(r.first == N);
    } else if (validator.group() == "5") {
        inf.ensure(N <= 3000);
    } else if (validator.group() == "6") {
        // Bez papildus ierobežojumiem
    }

}
