#include <bits/stdc++.h>
#define fi first
#define se second
#define pb push_back
using namespace std;
int main()
{
    ios_base::sync_with_stdio(0);
    cin.tie(0);cout.tie(0);
//    ifstream cin("in.in");
    int n;
    cin >> n;
    vector<pair<int,int> > ve(n), con;
    for(int i = 0;i<n;i++)
    {
        int a;
        cin >> a;
        ve[i]={a,a};
    }
    bool more = 1;
    while(more)
    {
        more=0;
        for(int i = 1;i<ve.size();i++)
        {
            if(ve[i].se==ve[i-1].fi-1)//liekam i uz i-1
            {
                con.pb({ve[i].fi,ve[i-1].fi});
                ve[i-1].fi=ve[i].fi;
                ve.erase(ve.begin()+i);
                i--;
                more=1;
            }
            else if(ve[i].fi==ve[i-1].se+1)// liekam i-1 uz i
            {
                con.pb({ve[i-1].fi,ve[i].fi});
                ve[i-1].se=ve[i].se;
                ve.erase(ve.begin()+i);
                i--;
                more=1;
            }
        }
    }
    int l = -1, r = -2;
    for(auto x:ve)
        if(r-l<x.se-x.fi)
            l=x.fi,
            r=x.se;
    vector<pair<int,int> > ans;
    for(auto x:con)
        if(l<=min(x.fi,x.se)&&max(x.fi,x.se)<=r)
            ans.pb(x);
    cout << ans.size()+1 << " " << ans.size() << "\n";
    for(auto x:ans)
        cout << x.fi << " " << x.se << "\n";
    return 0;
}
