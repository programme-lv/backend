#include "testlib.h"
#include <bits/stdc++.h>

using namespace std;

const int MAXN = 505;

int dp[MAXN][MAXN];

void calc(int l, int r){
    if(l > r) return;
    if(l == r){
        dp[l][r] = l;
        return;
    }
    if(dp[l][r] != -1) return;
    dp[l][r]=(int)1e9+9;
    for(int c = l ; c <= r; c ++ ){
        calc(l, c - 1);
        calc(c + 1, r);
        dp[l][r]=min(dp[l][r],max(dp[l][c-1],dp[c+1][r]) + c);
    }
}

int main(int argc, char ** argv)
try {
	registerInteraction(argc, argv);

	cout.exceptions(ios_base::badbit | ios_base::failbit);

#ifdef SIGPIPE
	if (signal(SIGPIPE, SIG_IGN) == SIG_ERR) {
		throw std::system_error(errno, std::system_category(), "signal");
	}
#endif

	int n = inf.readInt(1, 500, "N");

	cout << n << endl;
    tout << n << endl;

    for(int i = 1; i <= n; i ++ ){
        for(int j = 1; j <= n; j ++ ){
            dp[i][j] = -1;
        }
    }
	int knownLeft = 1;
    int knownRight = n;

    int query_count = 0;
    int total_cost = 0;

    while(true) {
        
        int K = ouf.readInt(1,n,"query");
        tout << K << endl;

        query_count ++ ;
        total_cost += K;
        if(query_count > n){
            quitf(_wa, "Too many queries!");
        }
        if(total_cost > 3400){
            quitf(_wa, "Exceeded query cost!");
        }
        if(K < knownLeft){
            cout << 1 << endl;
            tout << 1 << endl;
        }
        else if(K > knownRight){
            cout << -1 << endl;
            tout << -1 << endl;
        }
        else{
            if(knownLeft == knownRight){
                cout << 0 << endl;
                tout << 0 << endl;
                quitf(_ok, "Guessed the position with %d coins", total_cost );
            }
            else if(K == knownLeft){
                cout << 1 << endl;
                tout << 1 << endl;
                knownLeft = K + 1;
            }
            else if(K == knownRight){
                cout << "-1" << endl;
                tout << "-1" << endl;
                knownRight = K - 1;
            }
            else{
                calc(knownLeft, K - 1);
                calc(K + 1, knownRight);
                if(dp[knownLeft][K - 1] > dp[K + 1][knownRight]){
                    cout << -1 << endl;
                    tout << -1 << endl;
                    knownRight = K - 1;
                }
                else{
                    cout << +1 << endl;
                    tout << +1 << endl;
                    knownLeft = K + 1;
                }
            }
        }
	}
} catch (const std::exception& e) {
	quit(_pe, e.what());
}
