Lai garantētu, ka jūsu programmas "vaicājumus" saņem vērtēšanas sistēma,
jums ir "jāsinhronizē" ( *flush* ) izvaddatu plūsma ( *stdout* ) pēc katra vaicājuma.

```json
{
    "component": "table",
    "cols": [
        {"header": "Valoda","width": "1%"},
        {"header": "Piemērs"}
    ],
    "data": [
        ["C++","`std::cout << something << std::endl;` ... \"std::endl\" nodrošina sinhronizāciju"],
        ["Go","`fmt.Println(something)` ... standarta datu plūsma nav īpaši jāsinhronizē"],
        ["Java","`System.out.println(something);\nSystem.out.flush();`"],
        ["Pascal","`writeln(something);\nflush(output);`"],
        ["Python","`print(something, flush=True)`"]
    ]
}
```

Ja testa ietvaros tiks pārsniegts maksimāli atļautais vaicājumu skaits, tā statuss pēc testēšanas būs "Nepareiza atbilde".
