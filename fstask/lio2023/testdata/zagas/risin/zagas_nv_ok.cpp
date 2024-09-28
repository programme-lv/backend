#include <iostream>
#include <algorithm>
using namespace std;

int n, a[100005], b[2][100005];

int main()
{
    while(cin >> n)
    {
        for (int i = 0; i < n; i++) scanf("%d", &a[i]);
        sort(a, a+n);
        for (int i = 0; i < n; i += 2) {
            b[0][i] = a[i/2];
            if (i+1 < n) b[0][i+1] = a[i/2+(n+1)/2];
            b[1][n-1-i] = a[i/2+n/2];
            if (i+1 < n) b[1][n-1-(i+1)] = a[i/2];
        }
        int res[] = {a[n-1], a[n-1]};
        for (int i = 0; i+1 < n; i++) {
            res[0] = min(res[0], abs(b[0][i+1]-b[0][i]));
            res[1] = min(res[1], abs(b[1][i+1]-b[1][i]));
        }
        int best = 0;
        if (n%2 == 0 and res[1] > res[0]) best = 1;
        cout << res[best] << endl;
        for (int i = 0; i < n; i++) cout << b[best][i] << " ";
        cout << endl;
    }
}
