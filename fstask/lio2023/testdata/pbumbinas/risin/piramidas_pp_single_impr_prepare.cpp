#include <bits/stdc++.h>

using namespace std;
using ll = long long;

const int MAXC = 1000000;
const int SIDE = MAXC * 2 + 9;
int a[SIDE * 2];

struct ST {
    int l, r;
    int lC, rC;
    int val;
    bool stop_r, stop_l;
}s[MAXC * 8];

int cntT = 0;

void update(int it)
{
    s[it].stop_l = s[s[it].lC].stop_l || s[s[it].rC].stop_l;
    s[it].stop_r = s[s[it].lC].stop_r || s[s[it].rC].stop_r;
}

int initst(int l, int r)
{
    int it = ++cntT;
    s[it].l = l;
    s[it].r = r;
    s[it].stop_r = s[it].stop_l = true;
    s[it].val = 0;

    if (l < r)
    {
        int mid = l + (r - l) / 2;
        s[it].lC = initst(l, mid);
        s[it].rC = initst(mid + 1, r);
        update(it);
    }
    else
    {
        int p0 = a[SIDE + l];
        int pl = a[SIDE + l - 1];
        int pr = a[SIDE + l + 1];
        s[it].val = p0;
        s[it].stop_r = p0 <= pr;
        s[it].stop_l = pl >= p0;
    }
    return it;
}

bool check_condition(int it, bool stop_r)
{
    if (stop_r)
    {
        return s[it].stop_r;
    }
    else
    {
        return s[it].stop_l;
    }
}

int NOT_FOUND = -2 * 1000'000'000;

int get_pos_stopr(int it, int l, int r)
{
    if (!s[it].stop_r || s[it].r < l || r < s[it].l)
    {
        return NOT_FOUND;
    }

    if (s[it].l == s[it].r)
    {
        return s[it].l;
    }

    int val = get_pos_stopr(s[it].lC, l, r);
    if (val != NOT_FOUND) {
        return val;
    }
    return get_pos_stopr(s[it].rC, l, r);
}

int get_pos_stopl(int it, int l, int r)
{
    if (!s[it].stop_l || s[it].r < l || r < s[it].l)
    {
        return NOT_FOUND;
    }

    if (s[it].l == s[it].r)
    {
        return s[it].l;
    }

    int val = get_pos_stopl(s[it].rC, l, r);
    if (val != NOT_FOUND) {
        return val;
    }
    return get_pos_stopl(s[it].lC, l, r);
}

void push(int it)
{
    if (s[it].val)
    {
        s[s[it].lC].val += s[it].val;
        s[s[it].rC].val += s[it].val;
        s[it].val = 0;
    }
}


int get_val_answ[3];
void get_val_triple(int it, int pos)
{
    if (s[it].r < pos - 1 || pos + 1 < s[it].l)
    {
        return;
    }

    if (s[it].l == s[it].r)
    {
        get_val_answ[s[it].l - pos + 1] = s[it].val;
        return;
    }

    push(it);
    get_val_triple(s[it].lC, pos);
    get_val_triple(s[it].rC, pos);
}

void update_triple(int it, int pos)
{
    if (s[it].r < pos - 1 || pos + 1 < s[it].l)
    {
        return;
    }

    if (s[it].l == s[it].r)
    {
        int off = s[it].l - pos + 1;
        if (off == 0)
        {
            s[it].stop_r = get_val_answ[0] <= get_val_answ[1];
        }
        else if (off == 1)
        {
            s[it].stop_r = get_val_answ[1] <= get_val_answ[2];
            s[it].stop_l = get_val_answ[0] >= get_val_answ[1];
        }
        else
        {
            s[it].stop_l = get_val_answ[1] >= get_val_answ[2];
        }
        return;
    }

    update_triple(s[it].lC, pos);
    update_triple(s[it].rC, pos);
    update(it);
}

void inc_range(int it, int l, int r)
{
    if (s[it].r < l || r < s[it].l)
    {
        return;
    }

    if (l <= s[it].l && s[it].r <= r)
    {
        s[it].val += 1;
        return;
    }

    push(it);
    inc_range(s[it].lC, l, r);
    inc_range(s[it].rC, l, r);
    // update(it); Not needed, not recalculated
}

void inc_val_range(int it, int l, int r)
{
    if (r < l)
    {
        return;
    }
    inc_range(it, l, r);
    get_val_triple(it, l);
    update_triple(it, l);

    get_val_triple(it, r);
    update_triple(it, r);
}

int get_val(int it, int pos)
{
    if (s[it].l == s[it].r)
    {
        return s[it].val;
    }
    push(it);
    if (pos <= s[s[it].lC].r)
        return get_val(s[it].lC, pos);
    return get_val(s[it].rC, pos);
}


int main()
{
    vector<pair<int, ll>> input;
    int n;
    scanf("%d", &n);
    input.reserve(n);
    for (; n > 0; n--)
    {
        int p;
        ll q;
        scanf("%d %lld", &p, &q);
        input.push_back({p, q});
    }

    int suffix_tasks = 0;
    while (suffix_tasks < input.size())
    {
        if (input[input.size() - 1 - suffix_tasks].second > 2000)
        {
            break;
        }
        suffix_tasks++;
    }

    int query = 0;
    {
        vector<int> l, r;

        l.reserve(1000000);
        r.reserve(1000000);
        for (; query + suffix_tasks < input.size(); query++)
        {
            int p = input[query].first;
            ll b = input[query].second;
            p += SIDE;

            int lastpos;
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
            printf("%d %d\n", lastpos - SIDE, a[lastpos]);
        }
    }

    if (suffix_tasks > 0)
    {
        int MAXS = 2 * MAXC;
        int root = initst(-2 * MAXC, 2 * MAXC);

        for (; query < input.size(); query++)
        {
            int p = input[query].first;
            ll q = input[query].second;

            int lastpos = p;
            while (q > 0)
            {
                int rpos = get_pos_stopr(root, p, MAXS);
                int cnt = rpos - p;
                if (cnt >= q)
                {
                    lastpos = rpos - q + 1;
                    inc_val_range(root, lastpos, rpos);
                    break;
                }
                inc_val_range(root, p + 1, rpos);
                q -= cnt;

                int lpos = get_pos_stopl(root, -MAXS, p);
                cnt = p - lpos + 1;
                if (cnt >= q)
                {
                    lastpos = lpos + q - 1;
                    inc_val_range(root, lpos, lastpos);
                    break;
                }
                inc_val_range(root, lpos, p);
                q -= cnt;
            }

            int lastval = get_val(root, lastpos);

            printf("%d %d\n", lastpos, lastval);
        }
    }
}
