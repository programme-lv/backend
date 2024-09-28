#include <bits/stdc++.h>

using namespace std;

using ll = long long;

const int MAXN = 100000;
const int PREF = 150000; // PREF**2 / 2 >= 10**10 . 10^5 queries with 10^5 balls

const int MAXC = 1 + PREF + MAXN + PREF;

int n;
int a[MAXC];

int main()
{
    scanf("%d", &n);
    for (int i = 1; i <= n; i++)
    {
        int tmp;
        scanf("%d", &tmp);
        a[i + PREF] = tmp;
    }

    int q;
    scanf("%d", &q);
    for (; q > 0; q--)
    {
        int pos;
        ll ballcnt;
        scanf("%d %lld", &pos, &ballcnt);
        pos += PREF;

        int lastpos = 0;
        int lasth = 0;

        while (ballcnt > 0)
        {
            int spos = pos;
            while (a[spos - 1] < a[spos] || a[spos + 1] < a[spos])
            {
                if (a[spos - 1] < a[spos])
                    spos -= 1;
                else
                    spos += 1;
            }

            lasth = a[spos] += 1;
            lastpos = spos;

            ballcnt--;
        }


        printf("%d %d\n", lastpos - PREF, lasth);
    }

}
