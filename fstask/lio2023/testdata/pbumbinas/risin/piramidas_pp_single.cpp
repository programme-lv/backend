#include <bits/stdc++.h>

using namespace std;
using ll = long long;

const int MAXC = 1000000;

const int MAXST = MAXC * 9;
struct ST {
    int l, r;
    int lC, rC;
    int val;
    bool stop_r, stop_l;
}s[MAXST];

int cntT = 0;

int initst(int l, int r)
{
    int it = ++cntT;
    assert(it < MAXST);
    s[it].l = l;
    s[it].r = r;
    s[it].stop_r = s[it].stop_l = true;
    s[it].val = 0;

    if (l < r)
    {
        int mid = l + (r - l) / 2;
        s[it].lC = initst(l, mid);
        s[it].rC = initst(mid + 1, r);
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

int get_pos_stopr(int it, int pos)
{
    if (!s[it].stop_r || s[it].r < pos)
    {
        return NOT_FOUND;
    }

    if (s[it].l == s[it].r)
    {
        return s[it].l;
    }

    int val = get_pos_stopr(s[it].lC, pos);
    if (val != NOT_FOUND) {
        return val;
    }
    return get_pos_stopr(s[it].rC, pos);
}

int get_pos_stopl(int it, int pos)
{
    if (!s[it].stop_l || pos < s[it].l)
    {
        return NOT_FOUND;
    }

    if (s[it].l == s[it].r)
    {
        return s[it].l;
    }

    int val = get_pos_stopl(s[it].rC, pos);
    if (val != NOT_FOUND) {
        return val;
    }
    return get_pos_stopl(s[it].lC, pos);
}

int get_val_answ[3];
void get_val_and_inc(int it, int pos)
{
    if (s[it].r < pos - 1 || pos + 1 < s[it].l)
    {
        return;
    }

    if (s[it].l == s[it].r)
    {
        if (s[it].l - (pos - 1) == 1)
        {
            s[it].val += 1;
        }
        get_val_answ[s[it].l - (pos - 1)] = s[it].val;
        return;
    }

    get_val_and_inc(s[it].lC, pos);
    get_val_and_inc(s[it].rC, pos);
}

void update(int it)
{
    s[it].stop_l = s[s[it].lC].stop_l || s[s[it].rC].stop_l;
    s[it].stop_r = s[s[it].lC].stop_r || s[s[it].rC].stop_r;
}

void update_triple(int it, int pos)
{
    if (s[it].r < pos - 1 || pos + 1 < s[it].l)
    {
        return;
    }

    if (s[it].l == s[it].r)
    {
        int off = s[it].l - (pos - 1);
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

int inc_val(int it, int pos)
{
    get_val_and_inc(it, pos);
    update_triple(it, pos);
    return get_val_answ[1];
}

int main()
{
    int n;
    scanf("%d", &n);

    int root = initst(-2 * MAXC, 2 * MAXC);

    for (; n > 0; n--)
    {
        int p;
        ll q;
        scanf("%d %lld", &p, &q);

        int lastpos = p;
        int lastval = 0;
        while (q > 0)
        {
            q--;

            int rpos = get_pos_stopr(root, p);
            if (rpos > p)
            {
                lastval = inc_val(root, rpos);
                lastpos = rpos;
                continue;
            }

            int lpos = get_pos_stopl(root, p);
            lastval = inc_val(root, lpos);
            lastpos = lpos;
        }
        printf("%d %d\n", lastpos, lastval);
    }
}

