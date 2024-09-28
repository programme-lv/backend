#include <bits/stdc++.h>
#pragma GCC optimize("O3,unroll-loops")
#pragma GCC target("avx2")

using namespace std;
using ll = int;
using ii = pair<ll,ll>;

const ll MAXN = 5e5;
ll a[MAXN], l[MAXN], r[MAXN];
ii t[MAXN], history[MAXN];
ll interval_idx[MAXN];
ll h_length[MAXN];
bool removed[MAXN];

int main() {
    ios_base::sync_with_stdio(false); cin.tie(0); cout.tie(0);
    ll N;
    cin>>N;
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
    cout<<mx_height<<' ';

    for(ll i=0;i<N;i++){
        if(removed[i]) continue;
        for(ll j=t[i].second;j<=t[i].first;j++)
            interval_idx[j]=i;
    }
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

    bool interesting[N]; memset(interesting,0,sizeof(interesting));
    for(ll i=t[res_idx].second;i<=t[res_idx].first;i++)
        interesting[i]=1;
    
    vector<ii> res_h;
    for(ll i=0;i<h_idx;i++){
        if(interesting[history[i].first]||interesting[history[i].second]){
            res_h.push_back(history[i]);
        }
    }
    cout<<res_h.size()<<'\n';
    for(auto x:res_h)
        cout<<x.first+1<<' '<<x.second+1<<'\n';
}