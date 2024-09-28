#include <bits/stdc++.h>

using namespace std;

const int MAXC = 1000000;
const int SIDE = MAXC * 30;

int n;
int a[SIDE * 2];

int main()
{
    scanf("%d", &n);

    for (int i = 0; i < n; i++)
    {
        int p; 
        long long b;
        scanf("%d %lld", &p, &b);
        p += SIDE;

        int lastpos = p;
        for (; b > 0; b--)
        {
            int pos = p;
            while (a[pos] > a[pos + 1])
            {
                pos++;
            }
            while (a[pos - 1] < a[pos])
            {
                pos--;
            }
            a[pos]++;
            lastpos = pos;
        }
        printf("%d %d\n", lastpos - SIDE, a[lastpos]);
    }
}
