#include <bits/stdc++.h>

using namespace std;
using ll = long long;
using ii = pair<ll,ll>;

int main() {
    ios_base::sync_with_stdio(false);
    cin.tie(NULL);

    ll n, m, k;
    cin>>n>>m>>k;
    vector<string> matrix;
    matrix.resize(n);
    for(ll i=0;i<n;i++)
        cin>>matrix[i];
    
    ii a_pos = {-1,-1}; // upper left (x,y)
    ii b_pos = {-1,-1};
    for(ll i=0;i<n;i++)
        for(ll j=0;j<m;j++)
            if(matrix[i][j]=='A')
                a_pos = {j,i};
            else if(matrix[i][j]=='B')
                b_pos = {j,i};

    vector<vector<ll>> xs;
    xs.resize(n);
    for(ll i=0;i<n;i++){
        xs[i].resize(m);
        for(ll j=0;j<m;j++){
            xs[i][j] = 0;
            if(i) xs[i][j] += xs[i-1][j];
            if(j) xs[i][j] += xs[i][j-1];
            if(i&&j) xs[i][j] -= xs[i-1][j-1];
            if(matrix[i][j]=='X') xs[i][j]++;
        }
    }

    vector<vector<bool>> ok;
    ok.resize(n);
    for(ll i=0;i<n;i++){
        ok[i].resize(m);
        for(ll j=0;j<m;j++){
            if(i+k>n || j+k>m) ok[i][j] = false;
            else{
                ll x = xs[i+k-1][j+k-1];
                if(j>0) x -= xs[i+k-1][j-1];
                if(i>0) x -= xs[i-1][j+k-1];
                if(j>0&&i>0) x += xs[i-1][j-1];
                ok[i][j] = (x<=((k*k)/2));
            }
        }
    }

    vector<vector<ll>> dist;
    dist.resize(n);
    for(ll i=0;i<n;i++){
        dist[i].resize(m);
        for(ll j=0;j<m;j++)
            dist[i][j] = -1;
    }

    queue<ii> q;
    q.push(a_pos);
    dist[a_pos.second][a_pos.first] = 0;

    const ll ox[4] = {1,0,-1,0};
    const ll oy[4] = {0,1,0,-1};
    while(!q.empty()){
        auto [x,y] = q.front();
        q.pop();
        for(ll i=0;i<4;i++){
            ll nx = x+ox[i];
            ll ny = y+oy[i];
            if(nx<0 || nx+k>m || ny<0 || ny+k>n)
                continue;
            if(ok[ny][nx] && dist[ny][nx]==-1){
                dist[ny][nx] = dist[y][x]+1;
                q.push({nx,ny});
            }
        }
    }
    
    const ll INF = 1e9;
    ll ans = INF;
    for(ll i=0;i<k;i++)
        for(ll j=0;j<k;j++){
            ll x = b_pos.first-i;
            ll y = b_pos.second-j;
            if(x<0 || x+k>m || y<0 || y+k>n)
                continue;
            if(ok[y][x] && dist[y][x]!=-1)
                ans = min(ans, dist[y][x]);
        }
    
    if(ans==INF) cout<<"-1\n";
    else cout<<ans<<"\n";
}