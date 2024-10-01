#include <iostream>
#include <queue>
using namespace std;

#define MAXN 1000005

int dr[] = {-1, 0, 1, 0};
int dc[] = {0, 1, 0, -1};

int N, M, K, rA, cA, rB, cB;
vector<char> a[MAXN], b[MAXN], visited[MAXN];
vector<int> cntX_row[MAXN];  // [row][start pos of sliding window len=K]


int solve()
{
    int rBmin = rB - K + 1, rBmax = rB, cBmin = cB - K + 1, cBmax = cB;
    int X_max_ok = K*K/2;

    for (int i = 0; i < N; i++) {
        b[i] = vector<char>(M);
        visited[i] = vector<char>(M, 0);
        cntX_row[i] = vector<int>(M);
    }

    for (int i = 0; i < N; i++) {
        cntX_row[i][0] = 0;
        for (int j = 0; j < K; j++) cntX_row[i][0] += int(a[i][j] == 'X');
        for (int j = K; j < M; j++) cntX_row[i][j-K+1] = cntX_row[i][j-K] - int(a[i][j-K] == 'X') + int(a[i][j] == 'X');
    }

    for (int j = 0; j < M; j++) {
        int cntX_sq = 0;
        for (int i = 0; i < K; i++) cntX_sq += cntX_row[i][j];
        b[0][j] = (cntX_sq <= X_max_ok);
        for (int i = K; i < N; i++) {
            cntX_sq -= cntX_row[i-K][j];
            cntX_sq += cntX_row[i][j];
            b[i-K+1][j] = (cntX_sq <= X_max_ok);
        }
    }

    queue<pair<pair<int, int>, int>> Q;
    Q.push(make_pair(make_pair(rA, cA), 0));
    visited[rA][cA] = true;
    while(!Q.empty()) {
        pair<pair<int, int>, int> state = Q.front();
        Q.pop();
        int r = state.first.first, c = state.first.second, d = state.second;
        if (rBmin <= r and r <= rBmax and cBmin <= c and c <= cBmax) return d;
        for (int k = 0; k < 4; k++) {
            int r_next = r + dr[k], c_next = c + dc[k];
            if (r_next < 0 or c_next < 0 or r_next > N-K or c_next > M-K) continue;
            if (!b[r_next][c_next]) continue;
            if (visited[r_next][c_next]) continue;
            visited[r_next][c_next] = true;
            Q.push(make_pair(make_pair(r_next, c_next), d+1));
        }
    }

    return -1;
}


int main()
{
    while(cin >> N >> M >> K) {
        for (int i = 0; i < N; i++) {
            a[i] = vector<char>(M);
            for (int j = 0; j < M; j++) {
                scanf(" %c\n", &a[i][j]);
                if (a[i][j] == 'A') {
                    rA = i;
                    cA = j;
                }
                if (a[i][j] == 'B') {
                    rB = i;
                    cB = j;
                }
            }
        }
        int res = solve();
        printf("%d\n", res);
    }
}
