namespace go example

struct Data {
    1: string text
}

service format_data {
    Data do_transfer(1:Data data),
    Data do_format(1:Data data),
    Data do_bracket(1:Data data)

}