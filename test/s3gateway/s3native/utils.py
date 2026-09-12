import uuid


def get_unique_id(length=10):
    """
    Return unique id of specified length
    Args:
        length(int): Length of unique id
    Return:
        str: unique id
    """
    return str(uuid.uuid4())[-length:]
