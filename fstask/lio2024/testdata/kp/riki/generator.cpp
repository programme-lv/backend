#include "testlib.h"
#include <iostream>
#include <vector>
#include <queue>

using namespace std;
using ii = pair<int,int>;

int attempts = 1;

int dist_square(ii a, ii b) {
    int dy = a.first - b.first;
    int dx = a.second - b.second;
    return dy*dy + dx*dx;
}

void travel(ii start, ii end, int N, int M, int K, vector<vector<bool>>& visited) {
    ii cur = start;
    while(cur != end) {
        visited[cur.first][cur.second] = true;
        vector<ii> next;
        const int dy[] = {0, 0, 1, -1};
        const int dx[] = {1, -1, 0, 0};
        for(int i = 0; i < 4; i++) {
            int ny = cur.first + dy[i];
            int nx = cur.second + dx[i];
            if(ny < 0 || ny+K > N || nx < 0 || nx+K > M) continue;
            if(visited[ny][nx]) continue;
            next.push_back(ii(ny, nx));
        }
        sort(next.begin(), next.end(), [&](ii a, ii b) {
            // return euclid distance to C squared
            return dist_square(a, end) < dist_square(b, end);
        });
        if(next.size() == 0 || dist_square(next[0], end) >= dist_square(cur, end)) {
            // no more path to end
            break;
        }
        cur = next[0];
    }
}

bool generateMaze(int N, int M, int K, bool OK, string type) {
    vector<vector<char>> maze(N, vector<char>(M, '.'));

    // generate a random maze
    for(int i = 0; i < N; i++)
        for(int j = 0; j < M; j++)
        {
            if(type!="sparse") {
                if(OK){
                if(rnd.next(0, 1) == 0)
                    maze[i][j] = 'X';
                }
                else{
                if(rnd.next(0,2)<2)
                    maze[i][j] = 'X';
                }
            }
            else {
                if (rnd.next(0, 8) == 0)
                    maze[i][j] = 'X';
            }
        }
    
    // select end point
    ii B = ii(rnd.next(0, N-1), rnd.next(0, M-1)); // (y, x)
    maze[B.first][B.second] = 'B';

    // select start point
    vector<ii> ok_points;
    for(int i = 0; i < N; i++)
        for(int j = 0; j < M; j++) {
            if(i+K>N || j+K>M) continue;
            ok_points.push_back(ii(i, j));
        }
    if(find(ok_points.begin(), ok_points.end(), B) != ok_points.end())
        ok_points.erase(find(ok_points.begin(), ok_points.end(), B));
    if(ok_points.size() == 0) return false;
    ii A = ok_points[rnd.next(0, (int)sqrt((int)ok_points.size()-1))];
    maze[A.first][A.second] = 'A';

    // select two intermiadiate travel points C and D
    // ok_points.erase(find(ok_points.begin(), ok_points.end(), A));
    if(ok_points.size() == 0) return false;
    ii C = ok_points[rnd.next(0, (int)ok_points.size()-1)];
    // ok_points.erase(find(ok_points.begin(), ok_points.end(), C));
    if(ok_points.size() == 0) return false;
    ii D = ok_points[rnd.next(0, (int)ok_points.size()-1)];

    vector<vector<bool>> visited(N, vector<bool>(M, false));
    // travel from A to C
    cerr << "Traveling from A to C" << endl;
    travel(A, C, N, M, K, visited);


    if(OK){
        // travel from C to D
        cerr << "Traveling from C to D" << endl;
        travel(C, D, N, M, K, visited);

        //T travel from D to B
        cerr << "Traveling from D to B" << endl;
        travel(D, B, N, M, K, visited);
    }

    // clear the path
    cerr << "Clearing the path" << endl;
    for(int i=0;i<N;i++){
        for(int j=0;j<M;j++){
            if(!visited[i][j]) continue;
            vector<ii> blocked;
            for(int y = i; y < i+K; y++){
                for(int x = j; x < j+K; x++){
                    if(y >= N || x >= M) continue;
                    if(maze[y][x] == 'X') blocked.push_back(ii(y, x));
                }
            }
            int to_remove = blocked.size()-(K*K/2);
            if(to_remove <= 0) continue;
            shuffle(blocked.begin(), blocked.end());
            for(int k = 0; k < to_remove; k++){
                maze[blocked[k].first][blocked[k].second] = '.';
            }
        }
    }


    // print the maze
    cout << N << " " << M << " " << K << endl;
    for (int y = 0; y < N; ++y) {
        for (int x = 0; x < M; ++x) {
            cout << maze[y][x];
        }
        cout << endl;
    }
    return true;
}

int main(int argc, char* argv[]) {
    registerGen(argc, argv, 1);

    int N = atoi(argv[1]);
    int M = atoi(argv[2]);
    int K = atoi(argv[3]);
    int OK = atoi(argv[4]);
    string type = argv[5];

    while(!generateMaze(N, M, K, OK, type)){
        attempts ++;
    }

    return 0;
}
