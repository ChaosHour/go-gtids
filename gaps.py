#!/usr/bin/env python3
# -*- coding: utf-8 -*-


def find_gaps(retrieved, executed):
    retrieved_ranges = retrieved.split(',')
    executed_ranges = executed.split(',')

    retrieved_set = set()
    for rr in retrieved_ranges:
        _, range_str = rr.split(':')
        if '-' in range_str:
            start, end = map(int, range_str.split('-'))
            retrieved_set.update(range(start, end + 1))
        else:
            retrieved_set.add(int(range_str))

    executed_set = set()
    for er in executed_ranges:
        _, range_str = er.split(':')
        if '-' in range_str:
            start, end = map(int, range_str.split('-'))
            executed_set.update(range(start, end + 1))
        else:
            executed_set.add(int(range_str))

    return retrieved_set - executed_set




if __name__ == "__main__":
    retrieved = input("Enter Retrieved_Gtid_Set: ")
    executed = input("Enter Executed_Gtid_Set: ")

    gaps = find_gaps(retrieved, executed)
    print("The gaps are:", sorted(gaps))

    # Get the UUID from the retrieved GTID
    uuid = retrieved.split(':')[0]

    # Print each gap in the format UUID:gap
    for gap in sorted(gaps):
        print(f"{uuid}:{gap}")

    # If there are gaps, print the range in the format UUID:start-end
    if gaps:
        print("\nGTID_SUBTRACT:")
        print(f"{uuid}:{min(gaps)}-{max(gaps)}")
