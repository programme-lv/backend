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

        int left_fast_jump = p;
        int right_fast_jump = p;
        int lastpos = p;
        int pos = p; // Remember position
        for (; b > 0; b--)
        {
            if (pos == p)
            {
                if (a[pos] > a[pos + 1])
                {
                    pos = right_fast_jump;
                    while (a[pos] > a[pos + 1])
                    {
                        pos++;
                    }
                    right_fast_jump = pos;
                }
                if (a[pos - 1] < a[pos])
                {
                    pos = left_fast_jump;
                    while (a[pos - 1] < a[pos])
                    {
                        pos--;
                    }
                    left_fast_jump = pos;
                }
            }
            a[pos]++;
            lastpos = pos;

            if (pos < p)
                pos++;
            if (pos > p)
                pos--;

        }
        printf("%d %d\n", lastpos - SIDE, a[lastpos]);
    }
}
