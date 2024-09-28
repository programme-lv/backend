#include <bits/stdc++.h>

using namespace std;

const int N = 505;
int dp[N][N];
int go[N][N];

void calc(int l, int r){
    if (l > r)
        return;
    if (l == r) {
        dp[l][r] = go[l][r] = l;
        return;
    }
    if(dp[l][r] != -1) return;
    dp[l][r] = (int)1e9 + 9;
    go[l][r] = l;
    for(int cut = l ; cut <= r; cut ++ ){
        calc(l, cut - 1);
        calc(cut + 1, r);
        int cur_cost = cut + max(dp[l][cut - 1], dp[cut + 1][r]);
        if(cur_cost < dp[l][r]){
            dp[l][r] = cur_cost;
            go[l][r] = cut;
        }
    }
}


int main() {
    int n;
    cin >> n;
    for (int i = 1; i <= n; i++) {
        for(int j = 1; j <= n; j ++ ){
            dp[i][j] = -1;
        }
    }
    calc(1, n);
    int lf = 1;
    int rf = n;
    int query;
    while(lf < rf){
        query = go[lf][rf];
        cout << query << endl;
        int res;
        cin >> res;
        if(res == -1){
            rf = query - 1;
        }
        else if(res == +1){
            lf = query + 1;
        }
        else{
            return 0;
        }
    }
    cout << lf << endl;
    int res;
    cin >> res;
    return 0;
}