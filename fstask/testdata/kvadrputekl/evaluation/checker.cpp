#include "testlib.h"
#include <bits/stdc++.h>
using namespace std;
const int N = 1e6+5;
int a[N], pos[N], dsu[N], mi[N], mx[N], l[N], r[N], sz[N];
int root(int u)
{
    return dsu[u]=(dsu[u]==u?u:root(dsu[u]));
}
void merg(int u, int v)
{
    u=root(u);
    v=root(v);
    mi[v]=min(mi[v],mi[u]);
    mx[v]=max(mx[v],mx[u]);
    l[v]=min(l[v],l[u]);
    r[v]=max(r[v],r[u]);
    sz[v]+=sz[u];
    dsu[u]=v;
}
int main(int argc, char *argv[])
{
    registerTestlibCmd(argc, argv);
    int n = inf.readInt();
    for(int i = 1;i<=n;i++)
        a[i] = inf.readInt(),
        pos[a[i]]=dsu[i]=l[i]=r[i]=i,
        mi[i]=mx[i]=a[i],
        sz[i]=1;
    int len = ouf.readInt(1,n,"torna augstums");
    int m = ouf.readInt(0,n-1,"gajienu skaits");
    int bestlen = ans.readInt();
    quitif((len<bestlen),_wa, "Eksiste tornis ar lielaku augstumu");
    quitif((len<bestlen),_fail, "Kluda! Dalibnieks ir atradis labaku risinajumu, neka zurija!");
    quitif((len<m+1),_wa, "Eksiste isaka gajienu virkne");
    quitif((len!=m+1),_wa,"Gajienu virkne nav pareiza");
    for(int i = 0;i<m;i++)
    {
        int u = pos[ouf.readInt(1,n,"torna augseja ripa")];
        int v = pos[ouf.readInt(1,n,"torna augseja ripa")];
        int ru = root(u), rv = root(v);
        quitif((ru==rv),_wa,"Elementiem jabut dazados tornos");
        quitif((a[u]!=mi[ru]||a[v]!=mi[rv]),_wa,"Elementiem jareprezente tornu augsejas ripas");
        quitif((r[ru]!=l[rv]-1&&r[rv]!=l[ru]-1),_wa,"Torniem jabut blakus");
        quitif((mx[ru]!=mi[rv]-1),_wa,"Drikst savienot tornus tikai ar secigiem elementiem");
        merg(u,v);
    }
    int mxsz = -1;
    for(int i = 1;i<=n;i++)
        mxsz=max(mxsz,sz[root(i)]);
    quitif((mxsz!=len),_wa,"Gajienu virkne neizveido maksimala izmera torni");
    quitf(_ok,"Pareizi");
}
