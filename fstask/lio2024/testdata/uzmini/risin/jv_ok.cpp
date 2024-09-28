#include <iostream>
#include <fstream>
#include <cstdio>
#include <iomanip>
#include <sstream>
#include <cstring>
#include <string>
#include <cmath>
#include <stack>
#include <list>
#include <queue>
#include <deque>
#include <set>
#include <map>
#include <vector>
#include <algorithm>
#include <numeric>
#include <utility>
#include <functional>
#include <limits>
using namespace std;

typedef long long ll;
typedef unsigned long long ull;
typedef unsigned int ui;
typedef pair<int,int> pii;
typedef pair<ll,ll> pll;
typedef vector<vector<int> > graph;

const double pi = acos(-1.0);

#define oned(a, x1, x2) { cout << #a << ":"; for(int _i = (x1); _i < (x2); _i++){ cout << " " << a[_i]; } cout << endl; }
#define twod(a, x1, x2, y1, y2) { cout << #a << ":" << endl; for(int _i = (x1); _i < (x2); _i++){ for(int _j = (y1); _j < (y2); _j++){ cout << (_j > y1 ? " " : "") << a[_i][_j]; } cout << endl; } }

#define mp make_pair
#define pb push_back
#define fst first
#define snd second

const int INF = 1000000000;
const int MAXN = 505;

int C[MAXN][MAXN], P[MAXN][MAXN];

int main() {	
	for(int d = 1; d < MAXN; d++) {
		for(int i = 1; i+d < MAXN; i++) {
			int j = i+d;
			C[i][j] = INF;
			for(int k = i; k < j; k++) {
				int curr = k+max(C[i][k],C[k+1][j]);
				if(curr < C[i][j]) {
					C[i][j] = curr;
					P[i][j] = k;
				}
			}
		}
	}
	
	int L = 1, R;
	cin >> R;
	R++;
	while(true) {
		int M = P[L][R];
		cout << M << endl;
		int A; cin >> A;
		if(A<0) {
			R = M;
		} else if(A>0) {
			L = M+1;
		} else {
			break;
		}
	}
}
