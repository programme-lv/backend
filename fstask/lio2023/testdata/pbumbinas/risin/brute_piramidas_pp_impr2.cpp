#include <bits/stdc++.h>

using namespace std;

const int MAXC = 1000000;
const int SIDE = MAXC * 30;

int n;
int a[SIDE * 2];

int main()
{
    scanf("%d", &n);

    vector<int> l, r;

    l.reserve(1000000);
    r.reserve(1000000);

    for (int i = 0; i < n; i++)
    {
        int p; 
        long long b;
        scanf("%d %lld", &p, &b);
        p += SIDE;

        int lastpos;
        if (b <= 2)
        {
            lastpos = p;
            int pos = p; // Remember position
            for (; b > 0; b--)
            {
                if (pos == p)
                {
                    while (a[pos] > a[pos + 1])
                    {
                        pos++;
                    }
                    while (a[pos - 1] < a[pos])
                    {
                        pos--;
                    }
                }
                a[pos]++;
                lastpos = pos;

                if (pos < p)
                    pos++;
                if (pos > p)
                    pos--;

            }
        }
        else
        {
            int lpos = p;
            int rpos = p;
            int tmplh = 0;
            int tmprh = 0;
            l.clear();
            r.clear();
            l.push_back(0); // Including top

            while (b > 0)
            {
                while (a[rpos] + tmprh > a[rpos + 1])
                {
                    rpos++;
                    tmprh = 0;
                    r.push_back(0);
                }

                if (r.size() > 0)
                {
                    r.back() += 1;
                }

                if (r.size() >= b)
                {
                    if (r.size() > b)
                    {
                        r[r.size() - b - 1] -= 1;
                    }
                    lastpos = rpos - b + 1;
                    break;
                }

                b -= r.size();

                while (a[lpos - 1] < a[lpos] + tmplh)
                {
                    lpos--;
                    tmplh = 0;
                    l.push_back(0);
                }

                if (l.size() > 0)
                {
                    l.back() += 1;
                }

                if (l.size() >= b)
                {
                    if (l.size() > b)
                    {
                        l[l.size() - b - 1] -= 1;
                    }
                    lastpos = lpos + b - 1;
                    break;
                }

                b -= l.size();

                tmplh += 1;
                tmprh += 1;
            }

            // Update values
            int dh = 0;
            while (r.empty() == false)
            {
                dh += r.back();
                r.pop_back();
                a[rpos] += dh;
                rpos--;
            }
            dh = 0;
            while (l.empty() == false)
            {
                dh += l.back();
                l.pop_back();
                a[lpos] += dh;
                lpos++;
            }

        }
        printf("%d %d\n", lastpos - SIDE, a[lastpos]);
    }
}
