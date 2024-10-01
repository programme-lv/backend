#include <bits/stdc++.h>

using namespace std;
using ll = long long;
using ii = pair<ll,ll>;

bool ok(ll y, ll x, ll n, ll m, ll k, vector<string>& matrix) {
    if(y < 0 || y+k > n || x < 0 || x+k > m) return false;
    ll count=0;
    for(ll i=0;i<k;i++){
        for(ll j=0;j<k;j++){
            if(matrix[y+i][x+j]=='X') count++;
        }
    }
    return count<=((k*k)/2);
}

int main() {
    ios_base::sync_with_stdio(false);
    cin.tie(NULL);

    ll n, m, k;
    cin>>n>>m>>k;

    vector<string> matrix; // grid of '.' and 'X'
    matrix.resize(n);

    for(ll i=0;i<n;i++) {
        cin>>matrix[i];
    }

    ii A = {-1,-1}, B = {-1,-1};
    for(ll i=0;i<n;i++){
        for(ll j=0;j<m;j++){
            if(matrix[i][j] == 'A') {
                A = {i,j};
            }
            if(matrix[i][j] == 'B') {
                B = {i,j};
            }
        }
    }
    assert(A.first != -1 && B.first != -1);

    assert(ok(A.first, A.second, n, m, k, matrix));

    vector<vector<ll>> dist(n, vector<ll>(m, -1));
    queue<ii> q;
    q.push(A);
    dist[A.first][A.second] = 0;

    const ll dx[] = {1,0,-1,0};
    const ll dy[] = {0,1,0,-1};
    while(!q.empty()){
        ii v = q.front();
        q.pop();
        for(ll i=0;i<4;i++){
            ll x = v.second + dx[i];
            ll y = v.first + dy[i];
            if(ok(y,x,n,m,k,matrix) && dist[y][x] == -1){
                dist[y][x] = dist[v.first][v.second] + 1;
                q.push({y,x});
            }
        }
    }

    const ll INF = 1e18;
    ll res = INF;
    for(ll i=0;i<k;i++){
        for(ll j=0;j<k;j++){
            ll y=B.first-i;
            ll x=B.second-j;
            if(y<0||x<0) continue;
            if(dist[y][x] != -1) {
                res = min(res, dist[y][x]);
            }
        }
    }
    if(res == INF) {
        cout<<"-1\n";
    } else {
        cout<<res<<"\n";
    }
}