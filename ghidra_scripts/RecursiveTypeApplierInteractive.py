# Ghidra script to recursively apply struct types to pointed-to memory (Interactive version)
# @category MemTools
# @menupath Tools.Recursive Type Applier (Interactive)

from ghidra.program.model.data import PointerDataType, StructureDataType, ArrayDataType
from ghidra.program.model.symbol import SourceType
from javax.swing import JOptionPane

def get_pointer_target_type(field_dt):
    """Extract the target type from a pointer data type."""
    if not isinstance(field_dt, PointerDataType):
        return None
    return field_dt.getDataType()

def is_struct_pointer(field_dt):
    """Check if field is a pointer to a struct (not char* or void*)."""
    if not isinstance(field_dt, PointerDataType):
        return False
    target = field_dt.getDataType()
    if target is None:
        return False
    if target.getName() in ["char", "void", "undefined"]:
        return False
    while isinstance(target, ArrayDataType):
        target = target.getDataType()
    return isinstance(target, StructureDataType)

def is_pointer_array(field_dt):
    """Check if field is an array of struct pointers."""
    if not isinstance(field_dt, ArrayDataType):
        return False
    elem_type = field_dt.getDataType()
    return is_struct_pointer(elem_type)

def read_pointer(address, ptr_size):
    """Read a pointer value from memory."""
    if ptr_size == 4:
        return toAddr(getInt(address) & 0xFFFFFFFF)
    else:
        return toAddr(getLong(address))

def apply_type_recursive(address, data_type, visited, ptr_size, depth=0, max_depth=100):
    """
    Recursively apply data_type at address, then follow all struct pointers.
    """
    if depth > max_depth:
        println("{}Max depth reached at {}".format("  " * depth, address))
        return 0

    addr_long = address.getOffset()

    if addr_long in visited:
        println("{}Cycle detected, skipping: {}".format("  " * depth, address))
        return 0

    if addr_long == 0:
        return 0

    visited.add(addr_long)

    actual_type = data_type
    while isinstance(actual_type, PointerDataType):
        actual_type = actual_type.getDataType()
    while isinstance(actual_type, ArrayDataType):
        actual_type = actual_type.getDataType()

    if not isinstance(actual_type, StructureDataType):
        println("{}Not a struct type: {}".format("  " * depth, actual_type))
        return 0

    clearListing(address, address.add(actual_type.getLength() - 1))

    try:
        createData(address, actual_type)
        println("{}Applied {} at {}".format("  " * depth, actual_type.getName(), address))
    except Exception as e:
        println("{}Failed to apply {} at {}: {}".format("  " * depth, actual_type.getName(), address, e))
        return 0

    count = 1

    for i in range(actual_type.getNumComponents()):
        component = actual_type.getComponent(i)
        field_dt = component.getDataType()
        field_offset = component.getOffset()
        field_name = component.getFieldName()

        if is_struct_pointer(field_dt):
            target_type = get_pointer_target_type(field_dt)
            field_addr = address.add(field_offset)

            try:
                ptr_value = read_pointer(field_addr, ptr_size)
                if ptr_value.getOffset() != 0:
                    println("{}Following pointer {} -> {}".format("  " * depth, field_name or "unnamed", ptr_value))
                    count += apply_type_recursive(ptr_value, target_type, visited, ptr_size, depth + 1, max_depth)
            except Exception as e:
                println("{}Failed to read pointer at {}: {}".format("  " * depth, field_addr, e))

        elif is_pointer_array(field_dt):
            # Handle arrays of struct pointers
            arr_type = field_dt
            elem_type = arr_type.getDataType()
            target_type = get_pointer_target_type(elem_type)
            arr_len = arr_type.getNumElements()
            elem_size = elem_type.getLength()

            println("{}Processing pointer array {} with {} elements".format("  " * depth, field_name or "unnamed", arr_len))

            for j in range(arr_len):
                elem_addr = address.add(field_offset + j * elem_size)
                try:
                    ptr_value = read_pointer(elem_addr, ptr_size)
                    if ptr_value.getOffset() != 0:
                        println("{}Following array[{}] pointer -> {}".format("  " * depth, j, ptr_value))
                        count += apply_type_recursive(ptr_value, target_type, visited, ptr_size, depth + 1, max_depth)
                except Exception as e:
                    println("{}Failed to read array pointer at {}: {}".format("  " * depth, elem_addr, e))

    return count

def apply_type_at_address_interactive():
    """Allow user to specify address and type name."""
    dtm = currentProgram.getDataTypeManager()

    addr_str = askString("Address", "Enter the address (hex, e.g., 0x12345678):")
    if addr_str is None:
        return

    try:
        if addr_str.startswith("0x") or addr_str.startswith("0X"):
            addr_str = addr_str[2:]
        address = toAddr(long(addr_str, 16))
    except:
        popup("Invalid address format")
        return

    type_name = askString("Type Name", "Enter the struct type name:")
    if type_name is None:
        return

    # Search for the type
    found_types = []
    for dt in dtm.getAllDataTypes():
        if dt.getName() == type_name:
            found_types.append(dt)

    if len(found_types) == 0:
        popup("Type '{}' not found in Data Type Manager".format(type_name))
        return
    elif len(found_types) > 1:
        println("Found {} types with name '{}', using first one".format(len(found_types), type_name))

    data_type = found_types[0]

    if not isinstance(data_type, StructureDataType):
        popup("Type must be a struct, got: {}".format(type(data_type).__name__))
        return

    ptr_size = currentProgram.getDefaultPointerSize()
    visited = set()

    # Apply at root
    clearListing(address, address.add(data_type.getLength() - 1))
    try:
        createData(address, data_type)
        println("Applied {} at {}".format(data_type.getName(), address))
    except Exception as e:
        popup("Failed to apply type at root: {}".format(e))
        return

    visited.add(address.getOffset())
    count = 0

    # Process children
    for i in range(data_type.getNumComponents()):
        component = data_type.getComponent(i)
        field_dt = component.getDataType()
        field_offset = component.getOffset()
        field_name = component.getFieldName()

        if is_struct_pointer(field_dt):
            target_type = get_pointer_target_type(field_dt)
            field_addr = address.add(field_offset)

            try:
                ptr_value = read_pointer(field_addr, ptr_size)
                if ptr_value.getOffset() != 0:
                    println("Following root pointer {} -> {}".format(field_name or "unnamed", ptr_value))
                    count += apply_type_recursive(ptr_value, target_type, visited, ptr_size, 1)
            except Exception as e:
                println("Failed to read pointer at {}: {}".format(field_addr, e))

        elif is_pointer_array(field_dt):
            arr_type = field_dt
            elem_type = arr_type.getDataType()
            target_type = get_pointer_target_type(elem_type)
            arr_len = arr_type.getNumElements()
            elem_size = elem_type.getLength()

            for j in range(arr_len):
                elem_addr = address.add(field_offset + j * elem_size)
                try:
                    ptr_value = read_pointer(elem_addr, ptr_size)
                    if ptr_value.getOffset() != 0:
                        println("Following root array[{}] pointer -> {}".format(j, ptr_value))
                        count += apply_type_recursive(ptr_value, target_type, visited, ptr_size, 1)
                except Exception as e:
                    println("Failed to read array pointer at {}: {}".format(elem_addr, e))

    println("Done! Applied {} additional types.".format(count))
    popup("Applied root type + {} child types".format(count))

def main():
    choice = askChoice("Mode", "Select operation mode:",
        ["From cursor (existing data)", "Interactive (specify address and type)"],
        "From cursor (existing data)")

    if choice == "Interactive (specify address and type)":
        apply_type_at_address_interactive()
        return

    # From cursor mode
    ptr_size = currentProgram.getDefaultPointerSize()
    println("Pointer size: {} bytes".format(ptr_size))

    if currentLocation is None:
        popup("Please position cursor on a data location")
        return

    address = currentLocation.getAddress()
    println("Starting address: {}".format(address))

    data = getDataAt(address)
    if data is None:
        popup("No data defined at current location.\nPlease apply a struct type first, or use Interactive mode.")
        return

    data_type = data.getDataType()
    println("Data type: {}".format(data_type.getName()))

    actual_type = data_type
    while isinstance(actual_type, PointerDataType):
        actual_type = actual_type.getDataType()

    if not isinstance(actual_type, StructureDataType):
        popup("Current data must be a struct type, got: {}".format(data_type.getName()))
        return

    visited = set()
    visited.add(address.getOffset())
    count = 0

    for i in range(actual_type.getNumComponents()):
        component = actual_type.getComponent(i)
        field_dt = component.getDataType()
        field_offset = component.getOffset()
        field_name = component.getFieldName()

        if is_struct_pointer(field_dt):
            target_type = get_pointer_target_type(field_dt)
            field_addr = address.add(field_offset)

            try:
                ptr_value = read_pointer(field_addr, ptr_size)
                if ptr_value.getOffset() != 0:
                    println("Following root pointer {} -> {}".format(field_name or "unnamed", ptr_value))
                    count += apply_type_recursive(ptr_value, target_type, visited, ptr_size, 1)
            except Exception as e:
                println("Failed to read pointer at {}: {}".format(field_addr, e))

        elif is_pointer_array(field_dt):
            arr_type = field_dt
            elem_type = arr_type.getDataType()
            target_type = get_pointer_target_type(elem_type)
            arr_len = arr_type.getNumElements()
            elem_size = elem_type.getLength()

            for j in range(arr_len):
                elem_addr = address.add(field_offset + j * elem_size)
                try:
                    ptr_value = read_pointer(elem_addr, ptr_size)
                    if ptr_value.getOffset() != 0:
                        println("Following root array[{}] pointer -> {}".format(j, ptr_value))
                        count += apply_type_recursive(ptr_value, target_type, visited, ptr_size, 1)
                except Exception as e:
                    println("Failed to read array pointer at {}: {}".format(elem_addr, e))

    println("Done! Applied {} types.".format(count))
    popup("Recursively applied {} struct types".format(count))

main()
